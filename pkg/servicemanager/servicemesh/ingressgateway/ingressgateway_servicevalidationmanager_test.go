/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgateway

import (
	"context"
	"errors"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

var (
	igName  = servicemeshapi.Name("my-ingressgateway")
	igName1 = servicemeshapi.Name("my-ingressgateway-1")
)

func Test_IngressGatewayValidateCreate(t *testing.T) {
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef   func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveMeshReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains only Mesh References",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only Mesh OCID",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains both Mesh Reference and Mesh OCID",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsNotUnique)},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec not contains OCID and MeshRef",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsEmpty)},
		},
		{
			name: "Referred Mesh is deleting",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					currentTime := metav1.Now()
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsDeleting)},
		},
		{
			name: "Referred Mesh not found",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceNotFound)},
		},
		{
			name: "TLS mode with missing server cert",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode:              servicemeshapi.IngressListenerTlsConfigModeTls,
									ServerCertificate: nil,
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "server certificate is missing"},
		},
		{
			name: "TLS mode with more than 1 server cert",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode: servicemeshapi.IngressListenerTlsConfigModeTls,
									ServerCertificate: &servicemeshapi.TlsCertificate{
										OciTlsCertificate:        &servicemeshapi.OciTlsCertificate{},
										KubeSecretTlsCertificate: &servicemeshapi.KubeSecretTlsCertificate{},
									},
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "cannot specify more than 1 certificate source"},
		},
		{
			name: "MUTUAL_TLS mode with missing server cert",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode:              servicemeshapi.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: &servicemeshapi.TlsCertificate{},
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "missing certificate info"},
		},
		{
			name: "MUTUAL_TLS mode with missing trusted ca bundle 1",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode: servicemeshapi.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: &servicemeshapi.TlsCertificate{
										OciTlsCertificate: &servicemeshapi.OciTlsCertificate{CertificateId: "ocid.dummycert"},
									},
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "client validation config is missing"},
		},
		{
			name: "MUTUAL_TLS mode with missing trusted ca bundle 2",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode: servicemeshapi.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: &servicemeshapi.TlsCertificate{
										OciTlsCertificate: &servicemeshapi.OciTlsCertificate{CertificateId: "ocid.dummycert"},
									},
									ClientValidation: &servicemeshapi.IngressHostClientValidationConfig{
										TrustedCaBundle: nil,
									},
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "trusted ca bundle is missing"},
		},
		{
			name: "MUTUAL_TLS mode with more than 1 trusted ca bundle",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name:      "testHost",
							Hostnames: []string{"bookinfo.com"},
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
								Tls: &servicemeshapi.IngressListenerTlsConfig{
									Mode: servicemeshapi.IngressListenerTlsConfigModeMutualTls,
									ServerCertificate: &servicemeshapi.TlsCertificate{
										OciTlsCertificate: &servicemeshapi.OciTlsCertificate{CertificateId: "ocid.dummycert"},
									},
									ClientValidation: &servicemeshapi.IngressHostClientValidationConfig{
										TrustedCaBundle: &servicemeshapi.CaBundle{
											OciCaBundle:        &servicemeshapi.OciCaBundle{},
											KubeSecretCaBundle: &servicemeshapi.KubeSecretCaBundle{},
										},
									},
								},
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "cannot specify more than 1 caBundle source"},
		},
		{
			name: "HTTP without hostnames",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name: "testHost",
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
								Port:     8080,
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "hostnames is mandatory for a host with HTTP or TLS_PASSTHROUGH listener"},
		},
		{
			name: "TLS_PASSTHROUGH without hostnames",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name: "testHost",
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolTlsPassthrough,
								Port:     8080,
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, "hostnames is mandatory for a host with HTTP or TLS_PASSTHROUGH listener"},
		},
		{
			name: "TCP without hostnames",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my-comp",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{{
							Name: "testHost",
							Listeners: []servicemeshapi.IngressGatewayListener{{
								Protocol: servicemeshapi.IngressGatewayListenerProtocolTcp,
								Port:     8080,
							}},
						}},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewIngressGatewayValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IG")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})

			if tt.fields.ResolveMeshReference != nil {
				resolver.EXPECT().ResolveMeshReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.ingressGateway)

			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGateway, tt.wantErr.reason)
			}

			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_IngressGatewayValidateUpdate(t *testing.T) {
	type args struct {
		ingressGateway    *servicemeshapi.IngressGateway
		oldIngressGateway *servicemeshapi.IngressGateway
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr expectation
	}{
		{
			name: "Update when state is not Active",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NotActiveOnUpdate)},
		},
		{
			name: "Update when ServiceMeshDependenciesActive state is unknown",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.DependenciesIsUnknownOnUpdate)},
		},
		{
			name: "Update when ServiceMeshConfigured state is unknown",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.UnknownStateOnUpdate)},
		},
		{
			name: "Update when mesh Ref is changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh-1",
							},
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsImmutable)},
		},
		{
			name: "Update when mesh OCID is changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my.mesh.id",
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsImmutable)},
		},
		{
			name: "Update when spec is not changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec name is not supplied in new IG whereas old IG has a spec name",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &igName,
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is supplied in new IG whereas old IG does not have a spec name",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Name:          &igName1,
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is changed from old IG's spec name",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &igName1,
						CompartmentId: "my.compartmentid",
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Name:          &igName,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is not supplied in both old and new IG",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when description is changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Description:   &igDescription1,
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Description:   &igDescription,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec is changed but compartmentId remains the same",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Hosts: []servicemeshapi.IngressGatewayHost{
							{
								Name:      "testHost",
								Hostnames: []string{"test.com"},
								Listeners: []servicemeshapi.IngressGatewayListener{{
									Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
									Port:     80,
								}},
							},
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both listeners and compartmentId is changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						Hosts: []servicemeshapi.IngressGatewayHost{
							{
								Name:      "testHost",
								Hostnames: []string{"test.com"},
								Listeners: []servicemeshapi.IngressGatewayListener{{
									Protocol: servicemeshapi.IngressGatewayListenerProtocolHttp,
									Port:     80,
								}},
							},
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both access logging and compartmentId is changed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						AccessLogging: &servicemeshapi.AccessLogging{
							IsEnabled: true,
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when status is added",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my.compartmentid",
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewaySpec{
						CompartmentId: "my.compartmentid",
					},
				},
			},
			wantErr: expectation{true, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewIngressGatewayValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IG")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})

			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.ingressGateway, tt.args.oldIngressGateway)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGateway, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_IngressGatewayValidateDelete(t *testing.T) {
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr expectation
	}{
		{
			name: "State is True",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "State is False",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "State is Unknown",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.UnknownStatusOnDelete)},
		},
		{
			name: "Active is Unknown and configured is False",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshConfigured,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewIngressGatewayValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IG")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.ingressGateway)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.ingressGateway, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}
