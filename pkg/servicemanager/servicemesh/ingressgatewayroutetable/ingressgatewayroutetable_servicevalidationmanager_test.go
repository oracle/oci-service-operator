/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewayroutetable

import (
	"context"
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
	igrtSpecName     = servicemeshapi.Name("my-igrt")
	igrtSpecName1    = servicemeshapi.Name("my-igrt-1")
	igrtDescription  = servicemeshapi.Description("This is Ingress Gateway Route Table")
	igrtDescription1 = servicemeshapi.Description("This is Ingress Gateway Route Table 1")
)

func Test_IngressGatewayRouteTableValidateCreate(t *testing.T) {
	type args struct {
		IngressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveIngressGatewayReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error)
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains only ingress gateway References",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only ingress gateway OCID",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec contains both ingress gateway OCID and Ref",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
							Id: "my-ig-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and IngressGatewayRef",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: nil,
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsEmpty)},
		},
		{
			name: "Spec contains both virtual service OCID and Ref",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-ocid",
											},
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
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and VirtualServiceRef",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: nil,
											},
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
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Spec contains only HTTP traffic route rule",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
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
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only TCP traffic route rule",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								TcpRoute: &servicemeshapi.TcpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
										},
									},
									IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
										Name: "testHost",
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only TLS_PASSTHROUGH traffic route rule",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								TlsPassthroughRoute: &servicemeshapi.TlsPassthroughIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
										},
									},
									IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
										Name: "testHost",
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains all traffic route rule types",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
										},
									},
									IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
										Name: "testHost",
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
								TcpRoute: &servicemeshapi.TcpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
										},
									},
									IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
										Name: "testHost",
									},
								},
								TlsPassthroughRoute: &servicemeshapi.TlsPassthroughIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.TrafficRouteRuleIsNotUnique)},
		},
		{
			name: "Spec contains empty traffic route rule",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-ocid",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.TrafficRouteRuleIsEmpty)},
		},
		{
			name: "Referred ingress gateway is deleting",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					currentTime := metav1.Now()
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return igRef, nil
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsDeleting)},
		},
		{
			name: "Spec contains deleting virtual service in route table rules",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
							},
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
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
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveIngressGatewayReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
					igRef := &servicemeshapi.IngressGateway{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return igRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					currentTime := metav1.Now()
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsDeleting)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewIngressGatewayRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			if tt.fields.ResolveIngressGatewayReference != nil {
				resolver.EXPECT().ResolveIngressGatewayReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveIngressGatewayReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.IngressGatewayRouteTable)

			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.IngressGatewayRouteTable, tt.wantErr.reason)
			}

			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_IngressGatewayRouteTableValidateUpdate(t *testing.T) {
	type args struct {
		IngressGatewayRouteTable    *servicemeshapi.IngressGatewayRouteTable
		oldIngressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Update when state is not Active",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
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
			name: "Update when ingress gateway Ref is changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig-1",
							},
						},
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-ig",
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
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsImmutable)},
		},
		{
			name: "Update when ingress gateway OCID is changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
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
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsImmutable)},
		},
		{
			name: "Spec contains both ingress gateway OCID and Ref",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-ocid",
											},
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
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains both virtual deployment OCID and Ref",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{},
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
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Update when spec is not changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec name is not supplied in new IGRT whereas old IGRT has a spec name",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &igrtSpecName,
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is supplied in new IGRT whereas old IGRT does not have a spec name",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &igrtSpecName1,
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is changed from old IGRT's spec name",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &igrtSpecName1,
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &igrtSpecName,
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is not supplied in both old and new IGRT",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when description is changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Description:   &igrtDescription1,
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						Description:   &igrtDescription,
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec is changed by not changing the compartment id",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
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
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both route rules and compartmentId is changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId-1",
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
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
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both name and compartmentId is changed",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId-1",
						Name:          &igrtSpecName,
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id-1",
						},
						CompartmentId: "my-compId",
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
			name: "Update when status is added",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						CompartmentId: "my-compId-1",
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
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						CompartmentId: "my-compId",
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains deleting virtual service in route table rules",
			args: args{
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
											},
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
					},
				},
				oldIngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					currentTime := metav1.Now()
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsDeleting)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewIngressGatewayRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}
			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.IngressGatewayRouteTable, tt.args.oldIngressGatewayRouteTable)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.IngressGatewayRouteTable, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_IngressGatewayRouteTableValidateDelete(t *testing.T) {
	type args struct {
		IngressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
				IngressGatewayRouteTable: &servicemeshapi.IngressGatewayRouteTable{
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
			meshValidator := NewIngressGatewayRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("IGRT")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.IngressGatewayRouteTable)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.IngressGatewayRouteTable, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}
