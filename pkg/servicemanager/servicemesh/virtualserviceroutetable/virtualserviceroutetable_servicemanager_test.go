/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualserviceroutetable

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sdkcommons "github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	path            = "/foo"
	grpcEnabled     = true
	grpcDisabled    = false
	vsrtDescription = servicemeshapi.Description("This is Virtual Service Route Table")
	timeNow         = time.Now()
)

func TestResolveDependencies(t *testing.T) {
	type fields struct {
		ResolveVirtualServiceIdAndName func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		ResolveVirtualDeploymentId     []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
	}
	type args struct {
		vsrt *servicemeshapi.VirtualServiceRouteTable
	}
	requestTimeout2000 := int64(2000)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *servicemeshapi.VirtualServiceRouteTable
		wantErr error
	}{
		{
			name: "dependencies with empty namespace",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name: "my-vs",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name: "my-vd-1",
												},
											},
											Weight: 50,
										},
									},
									Path:               &path,
									IsGrpc:             &grpcEnabled,
									PathType:           servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
									RequestTimeoutInMs: &requestTimeout2000,
								},
							},
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name: "my-vd-2",
												},
											},
											Weight: 50,
										},
									},
									Path:     &path,
									IsGrpc:   &grpcDisabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd2-id")
						return &vdId, nil
					},
				},
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name: "my-vs",
						},
					},
					RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
									{
										VirtualDeployment: &servicemeshapi.RefOrId{
											ResourceRef: &servicemeshapi.ResourceRef{
												Name: "my-vd-1",
											},
										},
										Weight: 50,
									},
								},
								Path:               &path,
								IsGrpc:             &grpcEnabled,
								PathType:           servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								RequestTimeoutInMs: &requestTimeout2000,
							},
						},
						{
							HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
									{
										VirtualDeployment: &servicemeshapi.RefOrId{
											ResourceRef: &servicemeshapi.ResourceRef{
												Name: "my-vd-2",
											},
										},
										Weight: 50,
									},
								},
								Path:     &path,
								IsGrpc:   &grpcDisabled,
								PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "dependencies with namespace",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name:      "my-vd-1",
													Namespace: "my-namespace1",
												},
											},
											Weight: 50,
										},
									},
									Path:               &path,
									IsGrpc:             &grpcEnabled,
									PathType:           servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
									RequestTimeoutInMs: &requestTimeout2000,
								},
							},
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name:      "my-vd-2",
													Namespace: "my-namespace2",
												},
											},
											Weight: 50,
										},
									},
									Path:     &path,
									IsGrpc:   &grpcDisabled,
									PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd2-id")
						return &vdId, nil
					},
				},
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name:      "my-vs",
							Namespace: "my-namespace",
						},
					},
					RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
									{
										VirtualDeployment: &servicemeshapi.RefOrId{
											ResourceRef: &servicemeshapi.ResourceRef{
												Name:      "my-vd-1",
												Namespace: "my-namespace1",
											},
										},
										Weight: 50,
									},
								},
								Path:               &path,
								IsGrpc:             &grpcEnabled,
								PathType:           servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								RequestTimeoutInMs: &requestTimeout2000,
							},
						},
						{
							HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
									{
										VirtualDeployment: &servicemeshapi.RefOrId{
											ResourceRef: &servicemeshapi.ResourceRef{
												Name:      "my-vd-2",
												Namespace: "my-namespace2",
											},
										},
										Weight: 50,
									},
								},
								Path:     &path,
								IsGrpc:   &grpcDisabled,
								PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "vsrt without namespace and VS without namespace",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name: "my-vs",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name:      "my-vd-1",
													Namespace: "my-namespace1",
												},
											},
											Weight: 100,
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("vs " + string(virtualServiceRef.Name) + ":" + virtualServiceRef.Namespace + " not found")
				},
				ResolveVirtualDeploymentId: nil,
			},
			want:    nil,
			wantErr: errors.New("vs my-vs: not found"),
		},
		{
			name: "vsrt without namespace and VD without namespace",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name: "my-vd-1",
												},
											},
											Weight: 100,
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						return nil, errors.New("vd " + string(virtualDeploymentRef.Name) + ":" + virtualDeploymentRef.Namespace + " not found")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("vd my-vd-1: not found"),
		},
		{
			name: "vsrt VS not found",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("vs " + string(virtualServiceRef.Name) + ":" + virtualServiceRef.Namespace + " not found")
				},
				ResolveVirtualDeploymentId: nil,
			},
			want:    nil,
			wantErr: errors.New("vs my-vs:my-namespace not found"),
		},
		{
			name: "vsrt VD not found",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name:      "my-vd-1",
													Namespace: "my-namespace1",
												},
											},
											Weight: 100,
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						return nil, errors.New("vd " + string(virtualDeploymentRef.Name) + ":" + virtualDeploymentRef.Namespace + " not found")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("vd my-vd-1:my-namespace1 not found"),
		},
		{
			name: "vsrt VS is not active",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("virtual service is not active yet")
				},
				ResolveVirtualDeploymentId: nil,
			},
			want:    nil,
			wantErr: errors.New("virtual service is not active yet"),
		},
		{
			name: "vsrt VD is not active",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-vs",
								Namespace: "my-namespace",
							},
						},
						RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
									Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &servicemeshapi.RefOrId{
												ResourceRef: &servicemeshapi.ResourceRef{
													Name:      "my-vd-1",
													Namespace: "my-namespace1",
												},
											},
											Weight: 100,
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						return nil, errors.New("virtual deployment is not active yet")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("virtual deployment is not active yet"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			testFramework := framework.NewFakeClientFramework(t)

			m := &ResourceManager{
				referenceResolver: resolver,
				client:            testFramework.K8sClient,
			}

			if tt.fields.ResolveVirtualServiceIdAndName != nil {
				resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceIdAndName)
			}
			if tt.fields.ResolveVirtualDeploymentId != nil {
				for _, resolveVirtualDeploymentId := range tt.fields.ResolveVirtualDeploymentId {
					resolver.EXPECT().ResolveVirtualDeploymentId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(resolveVirtualDeploymentId)
				}
			}

			err := m.ResolveDependencies(ctx, tt.args.vsrt, &manager.ResourceDetails{})
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, tt.args.vsrt)
			}
		})
	}
}

func TestGetVsrt(t *testing.T) {
	type fields struct {
		GetVsrt func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error)
	}
	tests := []struct {
		name    string
		fields  fields
		vsrt    *servicemeshapi.VirtualServiceRouteTable
		wantErr error
	}{
		{
			name: "valid sdk vsrt",
			fields: fields{
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return &sdk.VirtualServiceRouteTable{
						LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateActive,
						CompartmentId:  conversions.String("ocid1.vsrt.oc1.iad.1"),
					}, nil
				},
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceRouteTableId: "my-vsrt",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk vsrt not found",
			fields: fields{
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return nil, errors.New("vsrt not found")
				},
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceRouteTableId: "my-vsrt",
				},
			},
			wantErr: errors.New("vsrt not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(ctrl)

			testFramework := framework.NewFakeClientFramework(t)

			m := &ResourceManager{
				serviceMeshClient: meshClient,
				client:            testFramework.K8sClient,
			}

			if tt.fields.GetVsrt != nil {
				meshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVsrt)
			}

			err := m.GetResource(ctx, tt.vsrt, &manager.ResourceDetails{})
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateK8s(t *testing.T) {
	type args struct {
		vsrt    *servicemeshapi.VirtualServiceRouteTable
		sdkVsrt *sdk.VirtualServiceRouteTable
		oldVsrt *servicemeshapi.VirtualServiceRouteTable
	}
	tests := []struct {
		name     string
		args     args
		wantErr  error
		want     *servicemeshapi.VirtualServiceRouteTable
		response *servicemanager.OSOKResponse
	}{
		{
			name: "vsrt updated with status as true",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				sdkVsrt: &sdk.VirtualServiceRouteTable{
					VirtualServiceId: conversions.String("my-vs-id"),
					Id:               conversions.String("my-vsrt-id"),
					LifecycleState:   sdk.VirtualServiceRouteTableLifecycleStateActive,
				},
				oldVsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualServiceRouteTable{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualServiceRouteTable",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-vsrt",
					ResourceVersion: "2",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId:            "my-vs-id",
					VirtualServiceName:          "my-vs-name",
					VirtualServiceRouteTableId:  "my-vsrt-id",
					VirtualDeploymentIdForRules: make([][]api.OCID, 0),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  string(commons.Successful),
								Message: string(commons.ResourceActive),
							},
						},
					},
				},
			},
		},
		{
			name: "vsrt updates with status as false",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId:           "my-vs-id",
						VirtualServiceName:         "my-vs-name",
						VirtualServiceRouteTableId: "my-vsrt-id",
					},
				},
				sdkVsrt: &sdk.VirtualServiceRouteTable{
					Id:             conversions.String("my-vsrt-id"),
					Name:           conversions.String("my-vsrt"),
					LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateFailed,
				},
				oldVsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualServiceRouteTable{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualServiceRouteTable",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-vsrt",
					ResourceVersion: "2",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId:            "my-vs-id",
					VirtualServiceName:          "my-vs-name",
					VirtualServiceRouteTableId:  "my-vsrt-id",
					VirtualDeploymentIdForRules: make([][]api.OCID, 0),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionFalse,
								Reason:  string(commons.LifecycleStateChanged),
								Message: string(commons.ResourceFailed),
							},
						},
					},
				},
			},
		},
		{
			name: "vsrt updates with status as unknown",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId:            "my-vs-id",
						VirtualServiceName:          "my-vs-name",
						VirtualServiceRouteTableId:  "my-vsrt-id",
						VirtualDeploymentIdForRules: make([][]api.OCID, 0),
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
				sdkVsrt: &sdk.VirtualServiceRouteTable{
					Id:             conversions.String("my-vsrt-id"),
					Name:           conversions.String("my-vsrt"),
					LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateUpdating,
				},
				oldVsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
			want: &servicemeshapi.VirtualServiceRouteTable{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualServiceRouteTable",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-vsrt",
					ResourceVersion: "2",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId:            "my-vs-id",
					VirtualServiceName:          "my-vs-name",
					VirtualServiceRouteTableId:  "my-vsrt-id",
					VirtualDeploymentIdForRules: make([][]api.OCID, 0),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(commons.LifecycleStateChanged),
								Message: string(commons.ResourceUpdating),
							},
						},
					},
				},
			},
		},
		{
			name: "vsrt description updated",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
						Description: &vsrtDescription,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId:            "my-vs-id",
						VirtualServiceName:          "my-vs-name",
						VirtualServiceRouteTableId:  "my-vsrt-id",
						VirtualDeploymentIdForRules: make([][]api.OCID, 0),
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:  metav1.ConditionTrue,
									Reason:  string(commons.Successful),
									Message: string(commons.ResourceActive),
								},
							},
						},
					},
				},
				sdkVsrt: &sdk.VirtualServiceRouteTable{
					Id:             conversions.String("my-vsrt-id"),
					Name:           conversions.String("my-vsrt"),
					Description:    conversions.String("This is Virtual Service Route Table"),
					LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateActive,
				},
				oldVsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-vsrt",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
					Description: &vsrtDescription,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId:            "my-vs-id",
					VirtualServiceName:          "my-vs-name",
					VirtualServiceRouteTableId:  "my-vsrt-id",
					VirtualDeploymentIdForRules: make([][]api.OCID, 0),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  string(commons.Successful),
								Message: string(commons.ResourceActive),
							},
						},
					},
				},
			},
		},
		{
			name: "vsrt no update needed",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId:            "my-vs-id",
						VirtualServiceName:          "my-vs-name",
						VirtualServiceRouteTableId:  "my-vsrt-id",
						VirtualDeploymentIdForRules: make([][]api.OCID, 0),
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:  metav1.ConditionTrue,
									Reason:  string(commons.Successful),
									Message: string(commons.ResourceActive),
								},
							},
						},
					},
				},
				sdkVsrt: &sdk.VirtualServiceRouteTable{
					Id:             conversions.String("my-vsrt-id"),
					Name:           conversions.String("my-vsrt"),
					LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateActive,
				},
				oldVsrt: &servicemeshapi.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-vsrt",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-vsrt",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId:            "my-vs-id",
					VirtualServiceName:          "my-vs-name",
					VirtualServiceRouteTableId:  "my-vsrt-id",
					VirtualDeploymentIdForRules: make([][]api.OCID, 0),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  string(commons.Successful),
								Message: string(commons.ResourceActive),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")},
			}
			vsrtDetails := &manager.ResourceDetails{}
			vsrtDetails.VsrtDetails.SdkVsrt = tt.args.sdkVsrt
			dependencies := conversions.VSRTDependencies{
				VirtualServiceName: servicemeshapi.Name("my-vs-name"),
				VdIdForRules:       make([][]api.OCID, 0),
			}
			vsrtDetails.VsrtDetails.Dependencies = &dependencies
			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)
			err := f.K8sClient.Create(ctx, tt.args.vsrt)
			assert.NoError(t, err)
			response, err := m.UpdateK8s(ctx, tt.args.vsrt, vsrtDetails, false, false)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				if tt.response != nil {
					assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
				} else {
					opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
					assert.True(t, cmp.Equal(tt.want, tt.args.vsrt, opts), "diff", cmp.Diff(tt.want, tt.args.vsrt, opts))
				}
			}
		})
	}
}

func TestFinalize(t *testing.T) {
	m := &ResourceManager{}
	err := m.Finalize(context.Background(), nil)
	assert.NoError(t, err)
}

func TestOsokFinalize(t *testing.T) {
	type fields struct {
		DeleteVsrt func(ctx context.Context, vsrtId *api.OCID) error
	}
	tests := []struct {
		name    string
		fields  fields
		vsrt    *servicemeshapi.VirtualServiceRouteTable
		wantErr error
	}{
		{
			name: "sdk vsrt deleted",
			fields: fields{
				DeleteVsrt: func(ctx context.Context, vsrtId *api.OCID) error {
					return nil
				},
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceRouteTableId: "my-vsrt",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk vsrt deletion error",
			fields: fields{
				DeleteVsrt: func(ctx context.Context, vsrtId *api.OCID) error {
					return errors.New("vsrt not deleted")
				},
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceRouteTableId: "my-vsrt",
				},
			},
			wantErr: errors.New("vsrt not deleted"),
		},
		{
			name: "sdk vsrt id is empty",
			fields: fields{
				DeleteVsrt: nil,
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceRouteTableId: "",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(controller)

			testFramework := framework.NewFakeClientFramework(t)

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				client:            testFramework.K8sClient,
				serviceMeshClient: meshClient,
			}

			m := manager.NewServiceMeshServiceManager(testFramework.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.DeleteVsrt != nil {
				meshClient.EXPECT().DeleteVirtualServiceRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteVsrt)
			}

			_, err := m.Delete(ctx, tt.vsrt)
			assert.True(t, len(tt.vsrt.ObjectMeta.Finalizers) != 0)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		vsrt *servicemeshapi.VirtualServiceRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualServiceRouteTable
	}{
		{
			name: "virtual service route table active condition updated with service mesh client error",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
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
								Status:             metav1.ConditionUnknown,
								Reason:             "MissingParameter",
								Message:            "Missing Parameter in the body (opc-request-id: 12-35-89 )",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table active condition updated with service mesh client timeout",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
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
								Status:             metav1.ConditionUnknown,
								Reason:             "ConnectionError",
								Message:            "Request to service timeout",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vsrt).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.vsrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vsrt, opts), "diff", cmp.Diff(tt.want, tt.args.vsrt, opts))
		})
	}
}

func TestUpdateServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		vsrt *servicemeshapi.VirtualServiceRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualServiceRouteTable
	}{
		{
			name: "virtual service route table dependencies active condition updated with service mesh client error",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             "MissingParameter",
								Message:            "Missing Parameter in the body (opc-request-id: 12-35-89 )",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table dependencies active condition updated with service mesh client timeout",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             "ConnectionError",
								Message:            "Request to service timeout",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table dependencies active condition updated with empty error message",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(commons.Successful),
								Message:            string(commons.DependenciesResolved),
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table dependencies active condition updated with k8s error message",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-vs-id is not active yet"),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.DependenciesNotResolved),
								Message:            "my-vs-id is not active yet",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vsrt).Build()

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.vsrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vsrt, opts), "diff", cmp.Diff(tt.want, tt.args.vsrt, opts))
		})
	}
}

func TestUpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		vsrt *servicemeshapi.VirtualServiceRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualServiceRouteTable
	}{
		{
			name: "virtual service route table configured condition updated with service mesh client error",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionFalse,
								Reason:             "MissingParameter",
								Message:            "Missing Parameter in the body (opc-request-id: 12-35-89 )",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table configured condition updated with service mesh client timeout",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             "ConnectionError",
								Message:            "Request to service timeout",
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service route table configured condition updated with empty error message",
			args: args{
				vsrt: &servicemeshapi.VirtualServiceRouteTable{
					Spec: servicemeshapi.VirtualServiceRouteTableSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualServiceRouteTable{
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(commons.Successful),
								Message:            string(commons.ResourceConfigured),
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vsrt).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.vsrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vsrt, opts), "diff", cmp.Diff(tt.want, tt.args.vsrt, opts))
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	type fields struct {
		ResolveVirtualServiceIdAndName func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		ResolveVirtualDeploymentId     []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		GetVsrt                        func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error)
		GetVsrtNewCompartment          func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error)
		CreateVsrt                     func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error)
		UpdateVsrt                     func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error
		ChangeVsrtCompartment          func(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error
	}
	tests := []struct {
		name                string
		vsrt                *servicemeshapi.VirtualServiceRouteTable
		fields              fields
		times               int
		wantErr             error
		expectOpcRetryToken bool
		doNotRequeue        bool
	}{
		{
			name: "VSRT Create without error",
			vsrt: getVsrtSpec(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				UpdateVsrt:            nil,
				ChangeVsrtCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "VSRT Create with error",
			vsrt: getVsrtSpec(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error) {
					return nil, errors.New("error in creating vsrt")
				},
				ChangeVsrtCompartment: nil,
				UpdateVsrt:            nil,
			},
			times:   1,
			wantErr: errors.New("error in creating vsrt"),
		},
		{
			name: "VSRT Create with error and store the retry token",
			vsrt: getVsrtSpec(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
				ChangeVsrtCompartment: nil,
				UpdateVsrt:            nil,
			},
			times:               1,
			expectOpcRetryToken: true,
			wantErr:             errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name: "VSRT created without error and clear retry token",
			vsrt: getVsrtSpec(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable, opcRetryToken *string) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				ChangeVsrtCompartment: nil,
				UpdateVsrt:            nil,
			},
			times:               1,
			expectOpcRetryToken: false,
		},
		{
			name: "VSRT Change compartment without error",
			vsrt: getVsrtWithDiffCompartmentId(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: nil,
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				UpdateVsrt: nil,
				ChangeVsrtCompartment: func(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "VSRT Change compartment with error",
			vsrt: getVsrtWithDiffCompartmentId(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				CreateVsrt: nil,
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				UpdateVsrt: nil,
				ChangeVsrtCompartment: func(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error {
					return errors.New("error in changing vsrt compartmentId")
				},
			},
			times:   1,
			wantErr: errors.New("error in changing vsrt compartmentId"),
		},
		{
			name: "VSRT Update without error",
			vsrt: getVsrtWithStatus("/bar", grpcDisabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				CreateVsrt: nil,
				UpdateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error {
					return nil
				},
				ChangeVsrtCompartment: nil,
			},
			times:   2,
			wantErr: nil,
		},
		{
			name: "VSRT Update with error",
			vsrt: getVsrtWithStatus("/bar", grpcDisabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				CreateVsrt: nil,
				UpdateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error {
					return errors.New("error in updating vsrt")
				},
				ChangeVsrtCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in updating vsrt"),
		},
		{
			name: "Resolve dependencies error on create",
			vsrt: getVsrtSpec(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("error in resolving dependencies")
				},
				ResolveVirtualDeploymentId: nil,
				GetVsrt:                    nil,
				CreateVsrt:                 nil,
				UpdateVsrt:                 nil,
				ChangeVsrtCompartment:      nil,
			},
			times:   1,
			wantErr: errors.New("error in resolving dependencies"),
		},
		{
			name: "get sdk error",
			vsrt: getVsrtWithStatus(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return nil, errors.New("error in getting vsrt")
				},
				CreateVsrt:            nil,
				UpdateVsrt:            nil,
				ChangeVsrtCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in getting vsrt"),
		},
		{
			name: "sdk vsrt is deleted",
			vsrt: getVsrtWithStatus(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return &sdk.VirtualServiceRouteTable{
						LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateDeleted,
					}, nil
				},
				CreateVsrt:            nil,
				UpdateVsrt:            nil,
				ChangeVsrtCompartment: nil,
			},
			times:        1,
			wantErr:      errors.New("virtual service route table in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "sdk vsrt is failed",
			vsrt: getVsrtWithStatus(path, grpcEnabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return &sdk.VirtualServiceRouteTable{
						LifecycleState: sdk.VirtualServiceRouteTableLifecycleStateFailed,
					}, nil
				},
				CreateVsrt:            nil,
				UpdateVsrt:            nil,
				ChangeVsrtCompartment: nil,
			},
			times:        1,
			wantErr:      errors.New("virtual service route table in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "VSRT change compartment with other spec fields",
			vsrt: getVsrtWithDiffCompartmentId("/bar", grpcDisabled),
			fields: fields{
				ResolveVirtualServiceIdAndName: func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceR := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("my-vs-name"),
					}
					return &virtualServiceR, nil
				},
				ResolveVirtualDeploymentId: []func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error){
					func(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
						vdId := api.OCID("my-vd1-id")
						return &vdId, nil
					},
				},
				GetVsrt: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.1"), nil
				},
				GetVsrtNewCompartment: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualServiceRouteTable, error) {
					return getSdkVsrt("ocid1.vsrt.oc1.iad.2"), nil
				},
				UpdateVsrt: func(ctx context.Context, vsrt *sdk.VirtualServiceRouteTable) error {
					return nil
				},
				ChangeVsrtCompartment: func(ctx context.Context, vsrtId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			times:   1,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(controller)
			resolver := meshMocks.NewMockResolver(controller)

			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.vsrt).Build()

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("vsrt")},
				serviceMeshClient: meshClient,
				client:            k8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			if tt.fields.ResolveVirtualServiceIdAndName != nil {
				resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceIdAndName).AnyTimes()
			}

			if tt.fields.ResolveVirtualDeploymentId != nil {
				for _, resolveVirtualDeploymentId := range tt.fields.ResolveVirtualDeploymentId {
					resolver.EXPECT().ResolveVirtualDeploymentId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(resolveVirtualDeploymentId).AnyTimes()
				}
			}

			if tt.fields.GetVsrt != nil {
				meshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVsrt).Times(tt.times)
			}

			if tt.fields.CreateVsrt != nil {
				meshClient.EXPECT().CreateVirtualServiceRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateVsrt)
			}

			if tt.fields.UpdateVsrt != nil {
				meshClient.EXPECT().UpdateVirtualServiceRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateVsrt)
			}

			if tt.fields.ChangeVsrtCompartment != nil {
				meshClient.EXPECT().ChangeVirtualServiceRouteTableCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeVsrtCompartment)
			}

			var response servicemanager.OSOKResponse
			for i := 0; i < tt.times; i++ {
				response, err = m.CreateOrUpdate(ctx, tt.vsrt, ctrl.Request{})
			}

			if tt.fields.GetVsrtNewCompartment != nil {
				meshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVsrtNewCompartment).Times(1)
				response, err = m.CreateOrUpdate(ctx, tt.vsrt, ctrl.Request{})
			}

			key := types.NamespacedName{Name: "my-vsrt", Namespace: "my-namespace"}
			curVsrt := &servicemeshapi.VirtualServiceRouteTable{}
			assert.NoError(t, k8sClient.Get(ctx, key, curVsrt))

			if tt.expectOpcRetryToken {
				assert.NotNil(t, curVsrt.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curVsrt.Status.OpcRetryToken)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, !tt.doNotRequeue, response.ShouldRequeue)
		})
	}
}

func getVsrtSpec(path string, isGrpc bool) *servicemeshapi.VirtualServiceRouteTable {
	return &servicemeshapi.VirtualServiceRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-vsrt",
		},
		Spec: servicemeshapi.VirtualServiceRouteTableSpec{
			CompartmentId: "ocid1.vsrt.oc1.iad.1",
			VirtualService: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name: "my-vs",
				},
			},
			RouteRules: []servicemeshapi.VirtualServiceTrafficRouteRule{
				{
					HttpRoute: &servicemeshapi.HttpVirtualServiceTrafficRouteRule{
						Destinations: []servicemeshapi.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeployment: &servicemeshapi.RefOrId{
									ResourceRef: &servicemeshapi.ResourceRef{
										Name: "my-vd-1",
									},
								},
								Weight: 100,
							},
						},
						Path:     &path,
						IsGrpc:   &isGrpc,
						PathType: servicemeshapi.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
					},
				},
			},
		},
	}
}

func getVsrtWithStatus(path string, isGrpc bool) *servicemeshapi.VirtualServiceRouteTable {
	vsrt := getVsrtSpec(path, isGrpc)
	vsrt.Status = servicemeshapi.ServiceMeshStatus{
		VirtualServiceRouteTableId: "my-vsrt-id",
		VirtualServiceId:           "my-vs-id",
	}
	return vsrt
}

func getVsrtWithDiffCompartmentId(path string, isGrpc bool) *servicemeshapi.VirtualServiceRouteTable {
	vsrt := getVsrtWithStatus(path, isGrpc)
	vsrt.Generation = 2
	newCondition := servicemeshapi.ServiceMeshCondition{
		Type: servicemeshapi.ServiceMeshActive,
		ResourceCondition: servicemeshapi.ResourceCondition{
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	vsrt.Status.Conditions = append(vsrt.Status.Conditions, newCondition)
	vsrt.Spec.CompartmentId = "ocid1.vsrt.oc1.iad.2"
	return vsrt
}

func getSdkVsrt(comaprtmentId string) *sdk.VirtualServiceRouteTable {
	return &sdk.VirtualServiceRouteTable{
		Id:               conversions.String("my-vsrt-id"),
		VirtualServiceId: conversions.String("my-vs-id"),
		LifecycleState:   sdk.VirtualServiceRouteTableLifecycleStateActive,
		CompartmentId:    conversions.String(comaprtmentId),
		TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
		TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
	}
}
