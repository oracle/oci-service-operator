/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"testing"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

var (
	vsrtName        = v1beta1.Name("my-vsrt")
	vsrtDescription = v1beta1.Description("This is Virtual Service Route Table")
)

func TestConvertCrdVsrtToSdkVsrt(t *testing.T) {
	type args struct {
		crdObj       *v1beta1.VirtualServiceRouteTable
		sdkObj       *sdk.VirtualServiceRouteTable
		dependencies *VSRTDependencies
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.VirtualServiceRouteTable
		wantErr    error
	}{
		{
			name: "No error case",
			args: args{
				crdObj: &v1beta1.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vsrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						Name:          &vsrtName,
						Description:   &vsrtDescription,
						CompartmentId: "my-compartment",
						VirtualService: v1beta1.RefOrId{
							Id: "my-vs-id",
						},
						RouteRules: []v1beta1.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 100,
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-newvd",
												},
											},
											Weight: 50,
											Port:   Port(8080),
										},
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 50,
											Port:   Port(8080),
										},
									},
									Path:     &path,
									IsGrpc:   &grpcDisabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
							{
								TcpRoute: &v1beta1.TcpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd-3",
												},
											},
											Weight: 100,
											Port:   Port(8080),
										},
									},
								},
							},
							{
								TlsPassthroughRoute: &v1beta1.TlsPassthroughVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd-3",
												},
											},
											Weight: 100,
											Port:   Port(8081),
										},
									},
								},
							},
						},
						Priority: Integer(1),
					},
					Status: v1beta1.ServiceMeshStatus{
						VirtualServiceRouteTableId: "my-vsrt-id",
					},
				},
				sdkObj: &sdk.VirtualServiceRouteTable{
					FreeformTags: map[string]string{"freeformTag2": "value2"},
					DefinedTags: map[string]map[string]interface{}{
						"definedTag2": {"key": "val"},
					},
				},
				dependencies: &VSRTDependencies{
					VirtualServiceId: "my-vs-id",
					VdIdForRules:     [][]api.OCID{{"my-vd-id"}, {"my-vd-id2", "my-vd-id"}, {"my-vd-3"}, {"my-vd-3"}},
				},
			},
			wantSDKObj: &sdk.VirtualServiceRouteTable{
				Id:               String("my-vsrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-vsrt"),
				Description:      String("This is Virtual Service Route Table"),
				VirtualServiceId: String("my-vs-id"),
				RouteRules: []sdk.VirtualServiceTrafficRouteRule{
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(100),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id2"),
								Weight:              Integer(50),
								Port:                Integer(8080),
							},
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(50),
								Port:                Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcDisabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
					sdk.TcpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-3"),
								Weight:              Integer(100),
								Port:                Integer(8080),
							},
						},
					},
					sdk.TlsPassthroughVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-3"),
								Weight:              Integer(100),
								Port:                Integer(8081),
							},
						},
					},
				},
				FreeformTags: map[string]string{
					"freeformTag1": "value1",
					"freeformTag2": "value2",
				},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
					"definedTag2": {"key": "val"},
				},
				Priority: Integer(1),
			},
		},
		{
			name: "No spec name case",
			args: args{
				crdObj: &v1beta1.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vsrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						VirtualService: v1beta1.RefOrId{
							Id: "my-vs-id",
						},
						RouteRules: []v1beta1.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 100,
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-newvd",
												},
											},
											Weight: 50,
										},
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 50,
											Port:   Port(8080),
										},
									},
									Path:     &path,
									IsGrpc:   &grpcDisabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
						Priority: Integer(1),
					},
					Status: v1beta1.ServiceMeshStatus{
						VirtualServiceRouteTableId: "my-vsrt-id",
					},
				},
				sdkObj: &sdk.VirtualServiceRouteTable{},
				dependencies: &VSRTDependencies{
					VirtualServiceId: "my-vs-id",
					VdIdForRules:     [][]api.OCID{{"my-vd-id"}, {"my-vd-id2", "my-vd-id"}},
				},
			},
			wantSDKObj: &sdk.VirtualServiceRouteTable{
				Id:               String("my-vsrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-namespace/my-vsrt"),
				VirtualServiceId: String("my-vs-id"),
				RouteRules: []sdk.VirtualServiceTrafficRouteRule{
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(100),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id2"),
								Weight:              Integer(50),
							},
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(50),
								Port:                Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcDisabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
				Priority: Integer(1),
			},
		},
		{
			name: "No Priority case",
			args: args{
				crdObj: &v1beta1.VirtualServiceRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-vsrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						Name:          &vsrtName,
						CompartmentId: "my-compartment",
						VirtualService: v1beta1.RefOrId{
							Id: "my-vs-id",
						},
						RouteRules: []v1beta1.VirtualServiceTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 100,
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
							{
								HttpRoute: &v1beta1.HttpVirtualServiceTrafficRouteRule{
									Destinations: []v1beta1.VirtualDeploymentTrafficRuleTarget{
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-newvd",
												},
											},
											Weight: 50,
										},
										{
											VirtualDeployment: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vd",
												},
											},
											Weight: 50,
											Port:   Port(8080),
										},
									},
									Path:     &path,
									IsGrpc:   &grpcDisabled,
									PathType: v1beta1.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						VirtualServiceRouteTableId: "my-vsrt-id",
					},
				},
				sdkObj: &sdk.VirtualServiceRouteTable{},
				dependencies: &VSRTDependencies{
					VirtualServiceId: "my-vs-id",
					VdIdForRules:     [][]api.OCID{{"my-vd-id"}, {"my-vd-id2", "my-vd-id"}},
				},
			},
			wantSDKObj: &sdk.VirtualServiceRouteTable{
				Id:               String("my-vsrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-vsrt"),
				VirtualServiceId: String("my-vs-id"),
				RouteRules: []sdk.VirtualServiceTrafficRouteRule{
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(100),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
					sdk.HttpVirtualServiceTrafficRouteRule{
						Destinations: []sdk.VirtualDeploymentTrafficRuleTarget{
							{
								VirtualDeploymentId: String("my-vd-id2"),
								Weight:              Integer(50),
							},
							{
								VirtualDeploymentId: String("my-vd-id"),
								Weight:              Integer(50),
								Port:                Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcDisabled),
						PathType: sdk.HttpVirtualServiceTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConvertCrdVsrtToSdkVsrt(tt.args.crdObj, tt.args.sdkObj, tt.args.dependencies)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}
