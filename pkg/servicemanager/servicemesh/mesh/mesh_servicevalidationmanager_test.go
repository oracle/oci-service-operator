/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mesh

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"testing"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

var (
	displayName        = servicemeshapi.Name("my-mesh-displayName")
	displayName1       = servicemeshapi.Name("my-mesh-displayName-old")
	certAuthorityOcid  = api.OCID("my-mesh-cert-ocid")
	certAuthorityOcid1 = api.OCID("my-mesh-cert-ocid-old")
)

func Test_ValidateCreateRequest(t *testing.T) {
	type args struct {
		mesh *servicemeshapi.Mesh
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name        string
		args        args
		expectation expectation
	}{
		{
			name: "create mesh successfully",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mesh",
					},
					Spec: servicemeshapi.MeshSpec{
						DisplayName: &displayName,
						Description: &meshDescription,
					},
				},
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.MeshSpec{
						DisplayName: &displayName,
						Description: &meshDescription,
					},
				},
			},
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.MetadataNameLengthExceeded),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewMeshValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})
			ctx := context.Background()
			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.mesh)

			if tt.expectation.reason != "" {
				tt.expectation.reason = errors.GetValidationErrorMessage(tt.args.mesh, tt.expectation.reason)
			}

			assert.Equal(t, tt.expectation.allowed, response.Allowed)
			assert.Equal(t, tt.expectation.reason, string(response.Result.Reason))
		})
	}
}

func Test_ValidateDeleteRequest(t *testing.T) {
	type args struct {
		mesh *servicemeshapi.Mesh
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name        string
		args        args
		expectation expectation
	}{
		{
			name: "State is True",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "State is False",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "State is Unknown",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.UnknownStatusOnDelete),
			},
		},
		{
			name: "Active is Unknown and configured is False",
			args: args{
				mesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewMeshValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})

			ctx := context.Background()
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.mesh)

			if tt.expectation.reason != "" {
				tt.expectation.reason = errors.GetValidationErrorMessage(tt.args.mesh, tt.expectation.reason)
			}

			assert.Equal(t, tt.expectation.allowed, response.Allowed)
			assert.Equal(t, tt.expectation.reason, string(response.Result.Reason))
		})
	}
}

func Test_ValidateUpdateRequest(t *testing.T) {
	type args struct {
		mesh    *servicemeshapi.Mesh
		oldMesh *servicemeshapi.Mesh
	}
	type fields struct {
		ResolveMeshId                        func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		ResolveVirtualServiceListByNamespace func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error)
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	tests := []struct {
		name        string
		args        args
		fields      fields
		expectation expectation
	}{
		{
			name: "Update when state is not Active",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldMesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.NotActiveOnUpdate),
			},
		},
		{
			name: "Update when ServiceMeshConfigured state is unknown",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldMesh: &servicemeshapi.Mesh{
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
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.UnknownStateOnUpdate),
			},
		},
		{
			name: "Update when spec is not changed",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when description is changed",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						Description:   &meshDescription1,
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						Description:   &meshDescription,
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
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
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when certificate authorities is changed but compartment id is not",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid1,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.CertificateAuthoritiesIsImmutable),
			},
		},
		{
			name: "Update when displayName and compartmentId is changed",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid1",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName1,
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
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					},
				},
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when mtls and compartmentId is changed",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid1",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName1,
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
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-1",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when Mtls is changed to lower mode and existing virtual services satisfy minimum",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive},
						DisplayName: &displayName1,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID("my-mesh-id"),
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
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:         "my-namespace",
							Name:              "my-virtualservice-1",
							DeletionTimestamp: nil,
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								ResourceRef: &servicemeshapi.ResourceRef{
									Namespace: "my-namespace",
									Name:      "my-mesh",
								},
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModePermissive,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualService2 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:         "my-namespace",
							Name:              "my-virtualservice-2",
							DeletionTimestamp: nil,
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								ResourceRef: &servicemeshapi.ResourceRef{
									Namespace: "my-namespace",
									Name:      "my-mesh",
								},
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeStrict},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModeStrict,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1, *virtualService2}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when Mtls is changed to higher mode and both existing virtual services do not satisfy",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName1,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID("my-mesh-id"),
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
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-1",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualService2 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-2",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModePermissive,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1, *virtualService2}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.MeshMtlsNotSatisfied),
			},
		},
		{
			name: "Update when Mtls is changed to higher mode and one of existing virtual services do not satisfy",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls:        &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						DisplayName: &displayName1,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID("my-mesh-id"),
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
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					},
				},
			},
			fields: fields{
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-1",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualService2 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-2",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeStrict},
						},
						Status: servicemeshapi.ServiceMeshStatus{
							VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
								Mode:          servicemeshapi.MutualTransportLayerSecurityModeStrict,
								CertificateId: conversions.OCID(certificateAuthorityId),
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1, *virtualService2}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.MeshMtlsNotSatisfied),
			},
		},
		{
			name: "Update when mtls does not satisfy and VS mode is interpreted from Spec as status is missing",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
						},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
						DisplayName: &displayName1,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID("my-mesh-id"),
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
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
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-1",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
							Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
								Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.MeshMtlsNotSatisfied),
			},
		},
		{
			name: "Update when mtls does not satisfy and VS mode is interpreted from old mesh status as status is missing",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 3},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
						},
						DisplayName: &displayName,
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
						DisplayName: &displayName1,
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID("my-mesh-id"),
						MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
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
				ResolveVirtualServiceListByNamespace: func(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
					virtualService1 := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-namespace",
							Name:      "my-virtualservice-1",
						},
						Spec: servicemeshapi.VirtualServiceSpec{
							Mesh: servicemeshapi.RefOrId{
								Id: api.OCID("my-mesh-id"),
							},
						},
					}
					virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1}}
					return virtualServiceList, nil
				},
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
			},
			expectation: expectation{
				allowed: false,
				reason:  string(meshCommons.MeshMtlsNotSatisfied),
			},
		},
		{
			name: "Update when status is added",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					},
				},
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
		{
			name: "Update when status is updated",
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
				oldMesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: "my.compartmentid",
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: certAuthorityOcid,
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
			},
			expectation: expectation{
				allowed: true,
				reason:  "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewMeshValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("Mesh")})

			ctx := context.Background()

			if tt.fields.ResolveVirtualServiceListByNamespace != nil {
				resolver.EXPECT().ResolveVirtualServiceListByNamespace(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceListByNamespace)
			}

			if tt.fields.ResolveMeshId != nil {
				resolver.EXPECT().ResolveMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshId).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.mesh, tt.args.oldMesh)

			if tt.expectation.reason != "" {
				tt.expectation.reason = errors.GetValidationErrorMessage(tt.args.mesh, tt.expectation.reason)
			}

			assert.Equal(t, tt.expectation.allowed, response.Allowed)
			assert.Equal(t, tt.expectation.reason, string(response.Result.Reason))
		})
	}
}
