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
	igrtName        = v1beta1.Name("my-igrt")
	igrtDescription = v1beta1.Description("This is Ingress Gateway Route Table")
	path            = "/foo"
	grpcEnabled     = true
	grpcDisabled    = false
)

func TestConvertCrdIgrtToSdkIgrt(t *testing.T) {
	type args struct {
		crdObj       *v1beta1.IngressGatewayRouteTable
		sdkObj       *sdk.IngressGatewayRouteTable
		dependencies *IGRTDependencies
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.IngressGatewayRouteTable
		wantErr    error
	}{
		{
			name: "Convert successfully Http",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						Description:   &igrtDescription,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
							},
						},
						Priority: Integer(0),
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				Description:      String("This is Ingress Gateway Route Table"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.HttpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
						Path:     String(path),
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
						},
					},
				},
				Priority:     Integer(0),
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully Tcp",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						Description:   &igrtDescription,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								TcpRoute: &v1beta1.TcpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
									},
								},
							},
						},
						Priority: Integer(0),
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				Description:      String("This is Ingress Gateway Route Table"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.TcpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
						},
					},
				},
				Priority:     Integer(0),
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully TlsPassthrough",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						Description:   &igrtDescription,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								TlsPassthroughRoute: &v1beta1.TlsPassthroughIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
										Port: Port(8080),
									},
								},
							},
						},
						Priority: Integer(0),
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				Description:      String("This is Ingress Gateway Route Table"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.TlsPassthroughIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
							Port: Integer(8080),
						},
					},
				},
				Priority:     Integer(0),
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully with multiple rules",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						Description:   &igrtDescription,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
							},
							{
								TcpRoute: &v1beta1.TcpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												Id: "my-vs-id-1",
											},
											Port: Port(9080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost1",
									},
								},
							},
							{
								TlsPassthroughRoute: &v1beta1.TlsPassthroughIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												Id: "my-vs-id-2",
											},
											Port: Port(9080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost2",
									},
								},
							},
						},
						Priority: Integer(0),
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}, {"my-vs-id-1"}, {"my-vs-id-2"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				Description:      String("This is Ingress Gateway Route Table"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.HttpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
						Path:     String(path),
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
						},
					},
					sdk.TcpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id-1"),
								Port:             Integer(9080),
							},
						},
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost1"),
						},
					},
					sdk.TlsPassthroughIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id-2"),
								Port:             Integer(9080),
							},
						},
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost2"),
						},
					},
				},
				Priority:     Integer(0),
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully with no priority",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.HttpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
						Path:     String(path),
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
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
			name: "Convert successfully with no Ingress gateway hosts",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &igrtName,
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-igrt"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.HttpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
						Path:     String(path),
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully with no spec name case",
			args: args{
				crdObj: &v1beta1.IngressGatewayRouteTable{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-igrt",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.IngressGatewayRouteTableSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						IngressGateway: v1beta1.RefOrId{
							Id: "my-ig-id",
						},
						RouteRules: []v1beta1.IngressGatewayTrafficRouteRule{
							{
								HttpRoute: &v1beta1.HttpIngressGatewayTrafficRouteRule{
									Destinations: []v1beta1.VirtualServiceTrafficRuleTarget{
										{
											VirtualService: &v1beta1.RefOrId{
												ResourceRef: &v1beta1.ResourceRef{
													Namespace: "my-namespace",
													Name:      "my-vs",
												},
												Id: "my-vs-id",
											},
											Port: Port(8080),
										},
									},
									IngressGatewayHost: &v1beta1.IngressGatewayHostRef{
										Name: "testHost",
									},
									Path:     &path,
									IsGrpc:   &grpcEnabled,
									PathType: v1beta1.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								},
							},
						},
						Priority: Integer(0),
					},
					Status: v1beta1.ServiceMeshStatus{
						IngressGatewayRouteTableId: "my-igrt-id",
					},
				},
				sdkObj: &sdk.IngressGatewayRouteTable{},
				dependencies: &IGRTDependencies{
					IngressGatewayId: "my-ig-id",
					VsIdForRules:     [][]api.OCID{{"my-vs-id"}},
				},
			},
			wantSDKObj: &sdk.IngressGatewayRouteTable{
				Id:               String("my-igrt-id"),
				CompartmentId:    String("my-compartment"),
				Name:             String("my-namespace/my-igrt"),
				IngressGatewayId: String("my-ig-id"),
				RouteRules: []sdk.IngressGatewayTrafficRouteRule{
					sdk.HttpIngressGatewayTrafficRouteRule{
						Destinations: []sdk.VirtualServiceTrafficRuleTarget{
							{
								VirtualServiceId: String("my-vs-id"),
								Port:             Integer(8080),
							},
						},
						IsGrpc:   Bool(grpcEnabled),
						PathType: sdk.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
						Path:     String(path),
						IngressGatewayHost: &sdk.IngressGatewayHostRef{
							Name: String("testHost"),
						},
					},
				},
				Priority:     Integer(0),
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConvertCrdIngressGatewayRouteTableToSdkIngressGatewayRouteTable(tt.args.crdObj, tt.args.sdkObj, tt.args.dependencies)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}
