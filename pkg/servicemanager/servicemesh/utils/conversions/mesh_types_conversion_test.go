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
	certificateAuthorityId = "my-certificate-authorityId"
	meshDisplayName        = v1beta1.Name("my-mesh")
	meshDescription        = v1beta1.Description("This is Mesh")
)

func TestConvertCrdMeshToSdkMesh(t *testing.T) {
	type args struct {
		crdObj *v1beta1.Mesh
		sdkObj *sdk.Mesh
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.Mesh
		wantErr    error
	}{
		{
			name: "Convert successfully",
			args: args{
				crdObj: &v1beta1.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.MeshSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						DisplayName:   &meshDisplayName,
						Description:   &meshDescription,
						CompartmentId: "my-compartment",
						CertificateAuthorities: []v1beta1.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId: "my-mesh-id",
					},
				},
				sdkObj: &sdk.Mesh{},
			},
			wantSDKObj: &sdk.Mesh{
				Id:            String("my-mesh-id"),
				CompartmentId: String("my-compartment"),
				DisplayName:   String("my-mesh"),
				Description:   String("This is Mesh"),
				CertificateAuthorities: []sdk.CertificateAuthority{
					{
						Id: String(certificateAuthorityId),
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully with no display name",
			args: args{
				crdObj: &v1beta1.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.MeshSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						CompartmentId: "my-compartment",
						CertificateAuthorities: []v1beta1.CertificateAuthority{
							{
								Id: *OCID(certificateAuthorityId),
							},
						},
						Mtls: &v1beta1.MeshMutualTransportLayerSecurity{
							Minimum: v1beta1.MutualTransportLayerSecurityModePermissive,
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId: "my-mesh-id",
					},
				},
				sdkObj: &sdk.Mesh{},
			},
			wantSDKObj: &sdk.Mesh{
				Id:            String("my-mesh-id"),
				CompartmentId: String("my-compartment"),
				DisplayName:   String("my-namespace/my-mesh"),
				CertificateAuthorities: []sdk.CertificateAuthority{
					{
						Id: String(certificateAuthorityId),
					},
				},
				Mtls: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Convert successfully with no MTLS",
			args: args{
				crdObj: &v1beta1.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.MeshSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo": "bar"},
							},
						},
						DisplayName:   &meshDisplayName,
						CompartmentId: "my-compartment",
						CertificateAuthorities: []v1beta1.CertificateAuthority{
							{
								Id: *OCID(certificateAuthorityId),
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId: "my-mesh-id",
					},
				},
				sdkObj: &sdk.Mesh{},
			},
			wantSDKObj: &sdk.Mesh{
				Id:            String("my-mesh-id"),
				CompartmentId: String("my-compartment"),
				DisplayName:   String("my-mesh"),
				CertificateAuthorities: []sdk.CertificateAuthority{
					{
						Id: String(certificateAuthorityId),
					},
				},
				FreeformTags: map[string]string{"freeformTag1": "value1"},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo": "bar"},
				},
			},
		},
		{
			name: "Patch crd tags with existing sdk tags",
			args: args{
				crdObj: &v1beta1.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
					Spec: v1beta1.MeshSpec{
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{"freeformTag1": "value1"},
							DefinedTags: map[string]api.MapValue{
								"definedTag1": {"foo1": "bar1"},
							},
						},
						DisplayName:   &meshDisplayName,
						CompartmentId: "my-compartment",
						CertificateAuthorities: []v1beta1.CertificateAuthority{
							{
								Id: *OCID(certificateAuthorityId),
							},
						},
					},
					Status: v1beta1.ServiceMeshStatus{
						MeshId: "my-mesh-id",
					},
				},
				sdkObj: &sdk.Mesh{
					FreeformTags: map[string]string{"freeformTag2": "value2"},
					DefinedTags: map[string]map[string]interface{}{
						"definedTag2": {"foo2": "bar2"},
					},
				},
			},
			wantSDKObj: &sdk.Mesh{
				Id:            String("my-mesh-id"),
				CompartmentId: String("my-compartment"),
				DisplayName:   String("my-mesh"),
				CertificateAuthorities: []sdk.CertificateAuthority{
					{
						Id: String(certificateAuthorityId),
					},
				},
				FreeformTags: map[string]string{
					"freeformTag1": "value1",
					"freeformTag2": "value2",
				},
				DefinedTags: map[string]map[string]interface{}{
					"definedTag1": {"foo1": "bar1"},
					"definedTag2": {"foo2": "bar2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConvertCrdMeshToSdkMesh(tt.args.crdObj, tt.args.sdkObj)
			assert.Equal(t, tt.wantSDKObj, tt.args.sdkObj)
		})
	}
}

func TestConvert_Crd_Mesh_MTls_To_Sdk_Mesh_Mtls(t *testing.T) {
	type args struct {
		crdObj *v1beta1.MeshMutualTransportLayerSecurity
	}
	tests := []struct {
		name       string
		args       args
		wantSDKObj *sdk.MeshMutualTransportLayerSecurity
		wantErr    error
	}{
		{
			name: "permissive case",
			args: args{
				crdObj: &v1beta1.MeshMutualTransportLayerSecurity{
					Minimum: v1beta1.MutualTransportLayerSecurityModePermissive,
				},
			},
			wantSDKObj: &sdk.MeshMutualTransportLayerSecurity{
				Minimum: sdk.MutualTransportLayerSecurityModePermissive,
			},
		},
		{
			name: "strict case",
			args: args{
				crdObj: &v1beta1.MeshMutualTransportLayerSecurity{
					Minimum: v1beta1.MutualTransportLayerSecurityModeStrict,
				},
			},
			wantSDKObj: &sdk.MeshMutualTransportLayerSecurity{
				Minimum: sdk.MutualTransportLayerSecurityModeStrict,
			},
		},
		{
			name: "disabled case",
			args: args{
				crdObj: &v1beta1.MeshMutualTransportLayerSecurity{
					Minimum: v1beta1.MutualTransportLayerSecurityModeDisabled,
				},
			},
			wantSDKObj: &sdk.MeshMutualTransportLayerSecurity{
				Minimum: sdk.MutualTransportLayerSecurityModeDisabled,
			},
		},
		{
			name: "unknown type case",
			args: args{
				crdObj: &v1beta1.MeshMutualTransportLayerSecurity{
					Minimum: "unknown",
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
			sdkObj, err := convertCrdMeshMTlsToSdkMeshMTls(tt.args.crdObj)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantSDKObj, sdkObj)
		})
	}
}

func TestConvert_Sdk_Mesh_Mtls_To_Crd_Mesh_Mtls(t *testing.T) {
	type args struct {
		sdkObj *sdk.MeshMutualTransportLayerSecurity
	}
	tests := []struct {
		name       string
		args       args
		wantCRDObj *v1beta1.MeshMutualTransportLayerSecurity
		wantErr    error
	}{
		{
			name: "permissive case",
			args: args{
				sdkObj: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
			},
			wantCRDObj: &v1beta1.MeshMutualTransportLayerSecurity{
				Minimum: v1beta1.MutualTransportLayerSecurityModePermissive,
			},
		},
		{
			name: "strict case",
			args: args{
				sdkObj: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModeStrict,
				},
			},
			wantCRDObj: &v1beta1.MeshMutualTransportLayerSecurity{
				Minimum: v1beta1.MutualTransportLayerSecurityModeStrict,
			},
		},
		{
			name: "disabled case",
			args: args{
				sdkObj: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModeDisabled,
				},
			},
			wantCRDObj: &v1beta1.MeshMutualTransportLayerSecurity{
				Minimum: v1beta1.MutualTransportLayerSecurityModeDisabled,
			},
		},
		{
			name: "unknown type case",
			args: args{
				sdkObj: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: "unknown",
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
			sdkObj, err := ConvertSdkMeshMTlsToCrdMeshMTls(tt.args.sdkObj)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCRDObj, sdkObj)
		})
	}
}
