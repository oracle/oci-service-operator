/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualserviceroutetable

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
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
	vsrtName         = servicemeshapi.Name("my-vsrt-name")
	vsrtName1        = servicemeshapi.Name("my-vsrt-name1")
	vsrtDescription1 = servicemeshapi.Description("This is Virtual Service Route Table 1")
)

func Test_VirtualServiceRouteTableValidateCreate(t *testing.T) {
	type args struct {
		VirtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef                func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualServiceReference    func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
		ResolveVirtualDeploymentReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr expectation
	}{
		{
			name: "Spec contains only virtual service References",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only virtual service OCID",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains both virtual service OCID and Ref",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
							Id: "my-vs-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and VirtualServiceRef",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Spec contains both virtual deployment OCID and Ref",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
												Id: "my-vd-ocid",
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and VirtualDeploymentRef",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsEmpty)},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
		{
			name: "Spec contains only HTTP route rule",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only TCP route rule",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								TcpRoute: &servicemeshapi.TcpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
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
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains only TLS_PASSTHROUGH route rule",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								TlsPassthroughRoute: &servicemeshapi.TlsPassthroughVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
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
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains empty traffic rule",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.TrafficRouteRuleIsEmpty)},
		},
		{
			name: "Spec contains all traffic route rules",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
								TcpRoute: &servicemeshapi.TcpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
								},
								TlsPassthroughRoute: &servicemeshapi.TlsPassthroughVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
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
			wantErr: expectation{false, string(commons.TrafficRouteRuleIsNotUnique)},
		},
		{
			name: "Referred Virtual Service is deleting",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
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
		{
			name: "Spec contains deleting virtual deployment in route table rules",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					currentTime := metav1.Now()
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsDeleting)},
		},
		{
			name: "Spec contains different destination ports",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Port: conversions.Port(8080),
										},
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Port: conversions.Port(8081),
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, "route rule destinations cannot have different ports"},
		},
		{
			name: "Spec contains different destination ports nil case",
			args: args{
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-ocid",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Port: conversions.Port(8080),
										},
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, "route rule destinations cannot have different ports"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewVirtualServiceRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VSRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualDeploymentReference != nil {
				resolver.EXPECT().ResolveVirtualDeploymentReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualDeploymentReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.VirtualServiceRouteTable)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.VirtualServiceRouteTable, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualServiceRouteTableValidateUpdate(t *testing.T) {
	type args struct {
		virtualServiceRouteTable    *servicemeshapi.VirtualServiceRouteTable
		oldVirtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable
	}
	type fields struct {
		ResolveResourceRef                func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveVirtualDeploymentReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
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
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
			name: "Update when virtual service Ref is changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs-1",
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-vs",
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsImmutable)},
		},
		{
			name: "Update when virtual service OCID is changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsImmutable)},
		},
		{
			name: "Spec contains both virtual deployment OCID and Ref",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
												Id: "my-vd-ocid",
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
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
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains both virtual deployment OCID and Ref",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
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
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsEmpty)},
		},
		{
			name: "Update when spec is not changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec name is not supplied in new VSRT whereas old VSRT has a spec name",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vsrtName,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is supplied in new VSRT whereas old VSRT does not have spec name",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Name:          &vsrtName1,
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when description is changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						Description:   &vsrtDescription1,
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						Description:   &vsrtDescription,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec name is changed from old VSRT's spec name",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						Name:          &vsrtName1,
						CompartmentId: "my-compId",
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						Name:          &vsrtName,
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.NameIsImmutable)},
		},
		{
			name: "Update when spec name is not supplied in both old and new VSRT",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when spec is changed by not changing the compartment id",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both route rules and compartmentId is changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when both spec Name and compartmentId is changed",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId-1",
						Name:          &vsrtName1,
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
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
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						CompartmentId: "my-compId",
					},
				},
			},
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains deleting virtual deployment in route table rules",
			args: args{
				virtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
						},
						CompartmentId: "my-compId",
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
				oldVirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id-1",
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
				ResolveVirtualDeploymentReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
					currentTime := metav1.Now()
					vdRef := &servicemeshapi.VirtualDeployment{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &currentTime,
						},
					}
					return vdRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualDeploymentReferenceIsDeleting)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewVirtualServiceRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VSRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualDeploymentReference != nil {
				resolver.EXPECT().ResolveVirtualDeploymentReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualDeploymentReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.virtualServiceRouteTable, tt.args.oldVirtualServiceRouteTable)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualServiceRouteTable, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_VirtualServiceRouteTableValidateDelete(t *testing.T) {
	type args struct {
		VirtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable
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
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
				VirtualServiceRouteTable: &servicemeshapi.VirtualServiceRouteTable{
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
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewVirtualServiceRouteTableValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VSRT")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VD")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.VirtualServiceRouteTable)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.VirtualServiceRouteTable, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}
