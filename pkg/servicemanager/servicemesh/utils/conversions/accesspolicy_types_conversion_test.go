/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"errors"
	"testing"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

var (
	apName        = v1beta1.Name("my-accesspolicy")
	apDescription = v1beta1.Description("This is Access Policy")
)

func TestConvert_CRD_AccessPolicy_To_SDK_AccessPolicy(t *testing.T) {
	type args struct {
		crdObj       *v1beta1.AccessPolicy
		sdkObj       *sdk.AccessPolicy
		dependencies *AccessPolicyDependencies
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.AccessPolicy
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &apName,
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
						Description: &apDescription,
						Rules: []v1beta1.AccessPolicyRule{
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									AllVirtualServices: &v1beta1.AllVirtualServices{},
								},
								Destination: v1beta1.TrafficTarget{
									VirtualService: &v1beta1.RefOrId{
										Id: "my-virtualservice-id",
									},
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{"destination": "my-virtualservice-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				Description:   String("This is Access Policy"),
				Rules: []sdk.AccessPolicyRule{
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.AllVirtualServicesAccessPolicyTarget{},
						Destination: sdk.VirtualServiceAccessPolicyTarget{
							VirtualServiceId: String("my-virtualservice-id"),
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
			name: "No spec name case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
						Description: &apDescription,
						Rules: []v1beta1.AccessPolicyRule{
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									AllVirtualServices: &v1beta1.AllVirtualServices{},
								},
								Destination: v1beta1.TrafficTarget{
									VirtualService: &v1beta1.RefOrId{
										Id: "my-virtualservice-id",
									},
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{"destination": "my-virtualservice-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-namespace/my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				Description:   String("This is Access Policy"),
				Rules: []sdk.AccessPolicyRule{
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.AllVirtualServicesAccessPolicyTarget{},
						Destination: sdk.VirtualServiceAccessPolicyTarget{
							VirtualServiceId: String("my-virtualservice-id"),
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
			name: "No access policy rules case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Name:          &apName,
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{"destination": "my-virtualservice-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				FreeformTags:  map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "ingress case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
						Rules: []v1beta1.AccessPolicyRule{
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									IngressGateway: &v1beta1.RefOrId{
										Id: "my-ingressgateway-id",
									},
								},
								Destination: v1beta1.TrafficTarget{
									VirtualService: &v1beta1.RefOrId{
										Id: "my-virtualservice-id",
									},
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{"destination": "my-virtualservice-id",
							"source": "my-ingressgateway-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-namespace/my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				Rules: []sdk.AccessPolicyRule{
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.IngressGatewayAccessPolicyTarget{
							IngressGatewayId: String("my-ingressgateway-id"),
						},
						Destination: sdk.VirtualServiceAccessPolicyTarget{
							VirtualServiceId: String("my-virtualservice-id"),
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
			name: "ingress to all case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
						Rules: []v1beta1.AccessPolicyRule{
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									IngressGateway: &v1beta1.RefOrId{
										Id: "my-ingressgateway-id",
									},
								},
								Destination: v1beta1.TrafficTarget{
									AllVirtualServices: &v1beta1.AllVirtualServices{},
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{
							"source": "my-ingressgateway-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-namespace/my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				Rules: []sdk.AccessPolicyRule{
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.IngressGatewayAccessPolicyTarget{
							IngressGatewayId: String("my-ingressgateway-id"),
						},
						Destination: sdk.AllVirtualServicesAccessPolicyTarget{},
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "external service case",
			args: args{
				crdObj: &v1beta1.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-accesspolicy",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.AccessPolicySpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						Mesh: v1beta1.RefOrId{
							Id: "my-mesh-id",
						},
						Rules: []v1beta1.AccessPolicyRule{
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									VirtualService: &v1beta1.RefOrId{
										Id: "my-vs-id",
									},
								},
								Destination: v1beta1.TrafficTarget{
									ExternalService: &v1beta1.ExternalService{HttpExternalService: &v1beta1.HttpExternalService{
										Hostnames: []string{"google.com"},
										Ports:     []v1beta1.Port{80},
									}},
								},
							},
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									AllVirtualServices: &v1beta1.AllVirtualServices{},
								},
								Destination: v1beta1.TrafficTarget{
									ExternalService: &v1beta1.ExternalService{HttpsExternalService: &v1beta1.HttpsExternalService{
										Hostnames: []string{"google.com"},
										Ports:     []v1beta1.Port{443},
									}},
								},
							},
							{
								Action: v1beta1.ActionTypeAllow,
								Source: v1beta1.TrafficTarget{
									VirtualService: &v1beta1.RefOrId{
										Id: "my-vs-id",
									},
								},
								Destination: v1beta1.TrafficTarget{
									ExternalService: &v1beta1.ExternalService{TcpExternalService: &v1beta1.TcpExternalService{
										IpAddresses: []string{"10.10.10.10/24"},
										Ports:       []v1beta1.Port{3306},
									}},
								},
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkObj: &sdk.AccessPolicy{},
				dependencies: &AccessPolicyDependencies{
					MeshId: "my-mesh-id",
					RefIdForRules: []map[string]api.OCID{
						{"source": "my-vs-id"},
						{},
						{"source": "my-vs-id"},
					},
				},
			},
			wantSDKObj: &sdk.AccessPolicy{
				Id:            String("my-accesspolicy-id"),
				CompartmentId: String("my-compartment"),
				Name:          String("my-namespace/my-accesspolicy"),
				MeshId:        String("my-mesh-id"),
				Rules: []sdk.AccessPolicyRule{
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.VirtualServiceAccessPolicyTarget{
							VirtualServiceId: String("my-vs-id"),
						},
						Destination: sdk.ExternalServiceAccessPolicyTarget{
							Hostnames: []string{"google.com"},
							Ports:     []int{80},
							Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttp,
						},
					},
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.AllVirtualServicesAccessPolicyTarget{},
						Destination: sdk.ExternalServiceAccessPolicyTarget{
							Hostnames: []string{"google.com"},
							Ports:     []int{443},
							Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttps,
						},
					},
					{
						Action: sdk.AccessPolicyRuleActionAllow,
						Source: sdk.VirtualServiceAccessPolicyTarget{
							VirtualServiceId: String("my-vs-id"),
						},
						Destination: sdk.ExternalServiceAccessPolicyTarget{
							IpAddresses: []string{"10.10.10.10/24"},
							Ports:       []int{3306},
							Protocol:    sdk.ExternalServiceAccessPolicyTargetProtocolTcp,
						},
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
			ConvertCrdAccessPolicyToSdkAccessPolicy(tt.args.crdObj, tt.args.sdkObj, tt.args.dependencies)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_Crd_AccessPolicyRule_To_Sdk_AccessPolicyRule(t *testing.T) {
	type args struct {
		crdObj       *v1beta1.AccessPolicyRule
		sdkObj       *sdk.AccessPolicyRule
		dependencies map[string]api.OCID
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.AccessPolicyRule
	}{
		{
			name: "normal case",
			args: args{
				crdObj: &v1beta1.AccessPolicyRule{
					Action: v1beta1.ActionTypeAllow,
					Source: v1beta1.TrafficTarget{
						AllVirtualServices: &v1beta1.AllVirtualServices{},
					},
					Destination: v1beta1.TrafficTarget{
						VirtualService: &v1beta1.RefOrId{
							Id: "my-virtualservice-id",
						},
					},
				},
				sdkObj:       &sdk.AccessPolicyRule{},
				dependencies: map[string]api.OCID{"destination": "my-virtualservice-id"},
			},
			wantSDKObj: &sdk.AccessPolicyRule{
				Action: sdk.AccessPolicyRuleActionAllow,
				Source: sdk.AllVirtualServicesAccessPolicyTarget{},
				Destination: sdk.VirtualServiceAccessPolicyTarget{
					VirtualServiceId: String("my-virtualservice-id"),
				},
			},
		},
		{
			name: "ingress case to vs case",
			args: args{
				crdObj: &v1beta1.AccessPolicyRule{
					Action: v1beta1.ActionTypeAllow,
					Source: v1beta1.TrafficTarget{
						IngressGateway: &v1beta1.RefOrId{
							Id: "my-ingressgateway-id",
						},
					},
					Destination: v1beta1.TrafficTarget{
						VirtualService: &v1beta1.RefOrId{
							Id: "my-virtualservice-id",
						},
					},
				},
				sdkObj: &sdk.AccessPolicyRule{},
				dependencies: map[string]api.OCID{"destination": "my-virtualservice-id",
					"source": "my-ingressgateway-id",
				},
			},
			wantSDKObj: &sdk.AccessPolicyRule{
				Action: sdk.AccessPolicyRuleActionAllow,
				Source: sdk.IngressGatewayAccessPolicyTarget{
					IngressGatewayId: String("my-ingressgateway-id"),
				},
				Destination: sdk.VirtualServiceAccessPolicyTarget{
					VirtualServiceId: String("my-virtualservice-id"),
				},
			},
		},
		{
			name: "ingress case to all vs case",
			args: args{
				crdObj: &v1beta1.AccessPolicyRule{
					Action: v1beta1.ActionTypeAllow,
					Source: v1beta1.TrafficTarget{
						IngressGateway: &v1beta1.RefOrId{
							Id: "my-ingressgateway-id",
						},
					},
					Destination: v1beta1.TrafficTarget{
						AllVirtualServices: &v1beta1.AllVirtualServices{},
					},
				},
				sdkObj: &sdk.AccessPolicyRule{},
				dependencies: map[string]api.OCID{
					"source": "my-ingressgateway-id",
				},
			},
			wantSDKObj: &sdk.AccessPolicyRule{
				Action: sdk.AccessPolicyRuleActionAllow,
				Source: sdk.IngressGatewayAccessPolicyTarget{
					IngressGatewayId: String("my-ingressgateway-id"),
				},
				Destination: sdk.AllVirtualServicesAccessPolicyTarget{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConvertCrdAccessPolicyRuleToSdkAccessPolicyRule(tt.args.crdObj, tt.args.sdkObj, tt.args.dependencies)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_Crd_TrafficTarget_To_Sdk_AccessPolicyTarget(t *testing.T) {
	type args struct {
		crdObj       *v1beta1.TrafficTarget
		dependencies api.OCID
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj sdk.AccessPolicyTarget
		wantErr    error
	}{
		{
			name: "all virtual services case",
			args: args{
				crdObj: &v1beta1.TrafficTarget{
					AllVirtualServices: &v1beta1.AllVirtualServices{},
				},
				dependencies: "",
			},
			wantSDKObj: sdk.AllVirtualServicesAccessPolicyTarget{},
		},
		{
			name: "virtual service case",
			args: args{
				crdObj: &v1beta1.TrafficTarget{
					VirtualService: &v1beta1.RefOrId{
						Id: "my-virtualservice-id",
					},
				},
				dependencies: "my-virtualservice-id",
			},
			wantSDKObj: sdk.VirtualServiceAccessPolicyTarget{
				VirtualServiceId: String("my-virtualservice-id"),
			},
		},
		{
			name: "ingress gateway case",
			args: args{
				crdObj: &v1beta1.TrafficTarget{
					IngressGateway: &v1beta1.RefOrId{
						Id: "my-ingressgateway-id",
					},
				},
				dependencies: "my-ingressgateway-id",
			},
			wantSDKObj: sdk.IngressGatewayAccessPolicyTarget{
				IngressGatewayId: String("my-ingressgateway-id"),
			},
		},
		{
			name: "external service case",
			args: args{
				crdObj: &v1beta1.TrafficTarget{
					ExternalService: &v1beta1.ExternalService{
						HttpExternalService: &v1beta1.HttpExternalService{
							Hostnames: []string{"*.test.com"},
							Ports:     []v1beta1.Port{80, 443},
						},
					},
				},
				dependencies: "my-virtualservice-id",
			},
			wantSDKObj: sdk.ExternalServiceAccessPolicyTarget{
				Hostnames: []string{"*.test.com"},
				Ports:     []int{80, 443},
				Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttp,
			},
		},
		{
			name: "external service no ports case",
			args: args{
				crdObj: &v1beta1.TrafficTarget{
					ExternalService: &v1beta1.ExternalService{
						HttpExternalService: &v1beta1.HttpExternalService{
							Hostnames: []string{"*.test.com"},
						},
					},
				},
				dependencies: "my-virtualservice-id",
			},
			wantSDKObj: sdk.ExternalServiceAccessPolicyTarget{
				Hostnames: []string{"*.test.com"},
				Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttp,
			},
		},
		{
			name: "unknown traffic target type",
			args: args{
				crdObj:       &v1beta1.TrafficTarget{},
				dependencies: "my-virtualservice-id",
			},
			wantErr: errors.New("unknown access policy target"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj, err := ConvertCrdTrafficTargetToSdkAccessPolicyTarget(tt.args.crdObj, tt.args.dependencies)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantSDKObj, sdkObj)
		})
	}
}
