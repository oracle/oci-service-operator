/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

var (
	vsName        = v1beta1.Name("my-virtualservice")
	vsDescription = v1beta1.Description("This is Virtual Service")
)

func Test_Convert_CRD_VirtualService_To_SDK_VirtualService(t *testing.T) {
	type args struct {
		crdObj *v1beta1.VirtualService
		sdkObj *sdk.VirtualService
		meshId api.OCID
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.VirtualService
	}{
		{
			name: "virtualservice with no default routing policy or mtls supplied",
			args: args{
				crdObj: &v1beta1.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-virtualservice",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:        &vsName,
						Description: &vsDescription,
						Hosts:       []string{"myhost"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkObj: &sdk.VirtualService{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.VirtualService{
				Id:            String("my-virtualservice-id"),
				Description:   String("This is Virtual Service"),
				CompartmentId: String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        String("my-mesh-id"),
				Name:          String("my-virtualservice"),
				FreeformTags:  map[string]string{},
				DefinedTags:   map[string]map[string]interface{}{},
			},
		},
		{
			name: "virtualservice with no spec name supplied",
			args: args{
				crdObj: &v1beta1.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-virtualservice",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []string{"myhost"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkObj: &sdk.VirtualService{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.VirtualService{
				Id:            String("my-virtualservice-id"),
				CompartmentId: String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        String("my-mesh-id"),
				Name:          String("my-namespace/my-virtualservice"),
				FreeformTags:  map[string]string{},
				DefinedTags:   map[string]map[string]interface{}{},
			},
		},
		{
			name: "virtualservice with supplied default routing policy",
			args: args{
				crdObj: &v1beta1.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-virtualservice",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name: &vsName,
						DefaultRoutingPolicy: &v1beta1.DefaultRoutingPolicy{
							Type: v1beta1.RoutingPolicyUniform,
						},
						Hosts: []string{"myhost"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkObj: &sdk.VirtualService{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.VirtualService{
				Id:            String("my-virtualservice-id"),
				CompartmentId: String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        String("my-mesh-id"),
				Name:          String("my-virtualservice"),
				DefaultRoutingPolicy: &sdk.DefaultVirtualServiceRoutingPolicy{
					Type: sdk.DefaultVirtualServiceRoutingPolicyTypeUniform,
				},
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "virtualservice with mtls supplied",
			args: args{
				crdObj: &v1beta1.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-virtualservice",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []string{"myhost"},
						Mtls: &v1beta1.CreateVirtualServiceMutualTransportLayerSecurity{
							Mode: v1beta1.MutualTransportLayerSecurityModePermissive,
						},
						Name: &vsName,
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkObj: &sdk.VirtualService{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.VirtualService{
				Id:            String("my-virtualservice-id"),
				CompartmentId: String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        String("my-mesh-id"),
				Name:          String("my-virtualservice"),
				Mtls:          &sdk.MutualTransportLayerSecurity{Mode: sdk.MutualTransportLayerSecurityModePermissive},
				FreeformTags:  map[string]string{},
				DefinedTags:   map[string]map[string]interface{}{},
			},
		},
		{
			name: "virtualservice with supplied tags",
			args: args{
				crdObj: &v1beta1.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-virtualservice",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: v1beta1.RefOrId{
							ResourceRef: &v1beta1.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:  &vsName,
						Hosts: []string{"myhost"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags:  map[string]api.MapValue{"definedTag1": {"foo": "bar"}},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkObj: &sdk.VirtualService{},
				meshId: api.OCID("my-mesh-id"),
			},
			wantSDKObj: &sdk.VirtualService{
				Id:            String("my-virtualservice-id"),
				CompartmentId: String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        String("my-mesh-id"),
				Name:          String("my-virtualservice"),
				FreeformTags:  map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = ConvertCrdVirtualServiceToSdkVirtualService(tt.args.crdObj, tt.args.sdkObj, &tt.args.meshId)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_Crd_MTls_To_Sdk_Mtls(t *testing.T) {
	type args struct {
		crdObj *v1beta1.CreateVirtualServiceMutualTransportLayerSecurity
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.MutualTransportLayerSecurity
		wantErr    error
	}{
		{
			name: "permissive case",
			args: args{
				crdObj: &v1beta1.CreateVirtualServiceMutualTransportLayerSecurity{
					Mode: v1beta1.MutualTransportLayerSecurityModePermissive,
				},
			},
			wantSDKObj: &sdk.MutualTransportLayerSecurity{
				Mode: sdk.MutualTransportLayerSecurityModePermissive,
			},
		},
		{
			name: "strict case",
			args: args{
				crdObj: &v1beta1.CreateVirtualServiceMutualTransportLayerSecurity{
					Mode: v1beta1.MutualTransportLayerSecurityModeStrict,
				},
			},
			wantSDKObj: &sdk.MutualTransportLayerSecurity{
				Mode: sdk.MutualTransportLayerSecurityModeStrict,
			},
		},
		{
			name: "disabled case",
			args: args{
				crdObj: &v1beta1.CreateVirtualServiceMutualTransportLayerSecurity{
					Mode: v1beta1.MutualTransportLayerSecurityModeDisabled,
				},
			},
			wantSDKObj: &sdk.MutualTransportLayerSecurity{
				Mode: sdk.MutualTransportLayerSecurityModeDisabled,
			},
		},
		{
			name: "unknown type case",
			args: args{
				crdObj: &v1beta1.CreateVirtualServiceMutualTransportLayerSecurity{
					Mode: "unknown",
				},
			},
			wantErr: errors.New("unknown MTLS mode type"),
		},
		{
			name: "null case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj, err := convertCrdVsMtlsToSdkVsMtls(tt.args.crdObj)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantSDKObj, sdkObj)
		})
	}
}

func TestConvert_Sdk_Vs_Mtls_To_Crd_Vs_Mtls(t *testing.T) {
	type args struct {
		sdkObj *sdk.MutualTransportLayerSecurity
	}
	tests := []struct {
		name       string
		args       args
		wantCRDObj *v1beta1.VirtualServiceMutualTransportLayerSecurity
		wantErr    error
	}{
		{
			name: "permissive case",
			args: args{
				sdkObj: &sdk.MutualTransportLayerSecurity{
					Mode:          sdk.MutualTransportLayerSecurityModePermissive,
					CertificateId: String(certificateAuthorityId),
				},
			},
			wantCRDObj: &v1beta1.VirtualServiceMutualTransportLayerSecurity{
				Mode:          v1beta1.MutualTransportLayerSecurityModePermissive,
				CertificateId: OCID(certificateAuthorityId),
			},
		},
		{
			name: "strict case",
			args: args{
				sdkObj: &sdk.MutualTransportLayerSecurity{
					Mode:          sdk.MutualTransportLayerSecurityModeStrict,
					CertificateId: String(certificateAuthorityId),
				},
			},
			wantCRDObj: &v1beta1.VirtualServiceMutualTransportLayerSecurity{
				Mode:          v1beta1.MutualTransportLayerSecurityModeStrict,
				CertificateId: OCID(certificateAuthorityId),
			},
		},
		{
			name: "disabled case",
			args: args{
				sdkObj: &sdk.MutualTransportLayerSecurity{
					Mode:          sdk.MutualTransportLayerSecurityModeDisabled,
					CertificateId: String(certificateAuthorityId),
				},
			},
			wantCRDObj: &v1beta1.VirtualServiceMutualTransportLayerSecurity{
				Mode:          v1beta1.MutualTransportLayerSecurityModeDisabled,
				CertificateId: OCID(certificateAuthorityId),
			},
		},
		{
			name: "unknown type case",
			args: args{
				sdkObj: &sdk.MutualTransportLayerSecurity{
					Mode:          "unknown",
					CertificateId: String(certificateAuthorityId),
				},
			},
			wantErr: errors.New("unknown MTLS mode type"),
		},
		{
			name: "certificate Id not present in response case",
			args: args{
				sdkObj: &sdk.MutualTransportLayerSecurity{
					Mode: sdk.MutualTransportLayerSecurityModeDisabled,
				},
			},
			wantCRDObj: &v1beta1.VirtualServiceMutualTransportLayerSecurity{
				Mode: v1beta1.MutualTransportLayerSecurityModeDisabled,
			},
		},
		{
			name: "null case",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkObj, err := ConvertSdkVsMtlsToCrdVsMtls(tt.args.sdkObj)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCRDObj, sdkObj)
		})
	}
}
