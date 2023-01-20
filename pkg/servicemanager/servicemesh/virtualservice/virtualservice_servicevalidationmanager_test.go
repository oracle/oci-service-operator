/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualservice

import (
	"context"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

var (
	vsName         = servicemeshapi.Name("my-vs-name")
	vsName1        = servicemeshapi.Name("my-vs-name1")
	vsDescription1 = servicemeshapi.Description("This is Virtual Service 1")
)

func Test_ValidateCreateRequest(t *testing.T) {
	type args struct {
		virtualService *servicemeshapi.VirtualService
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef   func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveMeshReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error)
		ResolveMeshRefById   func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error)
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
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
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
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Spec contains both Mesh OCID and MeshRef",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
							Id: "my-mesh-ocid",
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsNotUnique)},
		},
		{
			name: "Spec not contains OCID and MeshRef",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsEmpty)},
		},
		{
			name: "Referred Mesh is deleting",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Spec contains mtls that satisfies minimum when mesh resource ref is supplied",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
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
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains mtls that satisfies mesh minimum but mesh status is not yet updated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModeStrict,
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
								},
							},
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
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains mtls that satisfies default when mesh mtls is omitted and status is not yet updated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive,
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
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
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains mtls that does not satisfy default when mesh mtls is omitted and status is not yet updated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
								},
							},
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
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceMtlsNotSatisfied)},
		},
		{
			name: "Spec contains mtls that satisfies minimum when mesh Id is supplied",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "mesh-id",
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
			},
			fields: fields{
				ResolveMeshRefById: func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
					mesh := commons.MeshRef{
						Id:          "mesh-id",
						DisplayName: "mesh-name",
						Mtls:        servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					}
					return &mesh, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains mtls that does not satisfy minimum",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "mesh-id",
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
			},
			fields: fields{
				ResolveMeshRefById: func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
					mesh := commons.MeshRef{
						Id:          "mesh-id",
						DisplayName: "mesh-name",
						Mtls:        servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict},
					}
					return &mesh, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceMtlsNotSatisfied)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrller := gomock.NewController(t)
			defer ctrller.Finish()
			resolver := meshMocks.NewMockResolver(ctrller)
			meshValidator := NewVirtualServiceValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			if tt.fields.ResolveMeshReference != nil {
				resolver.EXPECT().ResolveMeshReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveMeshRefById != nil {
				resolver.EXPECT().ResolveMeshRefById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshRefById).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.virtualService)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualService, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_ValidateUpdateRequest(t *testing.T) {
	type args struct {
		virtualService    *servicemeshapi.VirtualService
		oldVirtualService *servicemeshapi.VirtualService
	}
	type fields struct {
		ResolveResourceRef                      func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveMeshReference                    func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error)
		ResolveMeshRefById                      func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error)
		ResolveVirtualServiceIdAndName          func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		ResolveHasVirtualDeploymentWithListener func(ctx context.Context, compartmentId *api.OCID, virtualServiceId *api.OCID) (bool, error)
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh-1",
							},
						},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my.mesh.id",
						},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update to delete hostname when there's virtual deployment has hostname and listeners",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Hosts:         []string{"hostname"},
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:     api.OCID("vs-id"),
						Name:   servicemeshapi.Name("vs"),
						MeshId: api.OCID("mesh-id"),
					}
					return &virtualServiceRef, nil
				},
				ResolveHasVirtualDeploymentWithListener: func(ctx context.Context, compartmentId *api.OCID, virtualServiceId *api.OCID) (bool, error) {
					return true, nil
				},
			},
			wantErr: expectation{false, "virtualservice hosts should not be empty when there's virtual deployment has listeners and hostname"},
		},
		{
			name: "Update to delete hostname when there's no virtual deployment has hostname and listeners",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Hosts:         []string{"hostname"},
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
				ResolveVirtualServiceIdAndName: func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:     api.OCID("vs-id"),
						Name:   servicemeshapi.Name("vs"),
						MeshId: api.OCID("mesh-id"),
					}
					return &virtualServiceRef, nil
				},
				ResolveHasVirtualDeploymentWithListener: func(ctx context.Context, compartmentId *api.OCID, virtualServiceId *api.OCID) (bool, error) {
					return false, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update to add hostname",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Hosts:         []string{"hostname"},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when spec is not changed",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when name is not supplied in new spec whereas old VS has a spec name",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &vsName,
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
			name: "Update when name is supplied in new spec whereas old VS does not have spec name",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Name:          &vsName1,
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when name is changed in spec from old VS's spec name",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &vsName1,
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &vsName,
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
			name: "Update when spec name is not supplied in both old and new VS",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Description:   &vsDescription,
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Description:   &vsDescription1,
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
			name: "Update when spec is changed by not changing the compartment id",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Hosts:         []string{"myhost"},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when both hosts and compartmentId is changed",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						Hosts:         []string{"myhost"},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when both spec Name and compartmentId is changed",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						Name:          &vsName1,
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Name:          &vsName,
						Hosts:         []string{"my-hostname"},
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
			name: "Update when both defaultRoutingPolicy and compartmentId is changed",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						DefaultRoutingPolicy: &servicemeshapi.DefaultRoutingPolicy{
							Type: servicemeshapi.RoutingPolicyUniform,
						},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "Update when both mtls and compartmentId is changed",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid1",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
						VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
							Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
							CertificateId: conversions.OCID(certificateAuthorityId),
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
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
							MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
								Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
							},
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when mtls satisfies minimum when mesh resource ref is supplied",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
						VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
							Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
							CertificateId: conversions.OCID(certificateAuthorityId),
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
							Namespace: "my-namespace",
							Name:      "my-mesh",
						},
						Spec: servicemeshapi.MeshSpec{
							CompartmentId: "my.compartmentid",
							CertificateAuthorities: []servicemeshapi.CertificateAuthority{
								{
									Id: "certAuthority-id",
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
							MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
								Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
							},
						},
					}
					return meshRef, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when mtls satisfies minimum when mesh id is supplied",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						CompartmentId: "my.compartmentid",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						CompartmentId: "my.compartmentid",
						Mtls:          &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
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
						VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					},
				},
			},
			fields: fields{
				ResolveMeshRefById: func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
					mesh := commons.MeshRef{
						Id:          "mesh-id",
						DisplayName: "mesh-name",
						Mtls:        servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
					}
					return &mesh, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when mtls does not satisfy minimum",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "mesh-id",
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModePermissive},
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "mesh-id",
						},
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: servicemeshapi.MutualTransportLayerSecurityModeStrict},
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
						VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModeStrict,
						},
					},
				},
			},
			fields: fields{
				ResolveMeshRefById: func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
					mesh := commons.MeshRef{
						Id:          "mesh-id",
						DisplayName: "mesh-name",
						Mtls:        servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict},
					}
					return &mesh, nil
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceMtlsNotSatisfied)},
		},
		{
			name: "Update when mtls is not supplied in new virtual service",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						CompartmentId: "my.compartmentid",
						Mtls: &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{
							Mode: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
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
				ResolveMeshRefById: func(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
					mesh := commons.MeshRef{
						Id:          "mesh-id",
						DisplayName: "mesh-name",
						Mtls: servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
						},
					}
					return &mesh, nil
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Update when status is added",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.VirtualServiceSpec{
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
			meshValidator := NewVirtualServiceValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})

			if tt.fields.ResolveMeshReference != nil {
				resolver.EXPECT().ResolveMeshReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveMeshRefById != nil {
				resolver.EXPECT().ResolveMeshRefById(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshRefById).AnyTimes()
			}

			if tt.fields.ResolveHasVirtualDeploymentWithListener != nil {
				resolver.EXPECT().ResolveHasVirtualDeploymentWithListener(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveHasVirtualDeploymentWithListener).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceIdAndName != nil {
				resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceIdAndName).AnyTimes()
			}
			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.virtualService, tt.args.oldVirtualService)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualService, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_ValidateDeleteRequest(t *testing.T) {
	type args struct {
		virtualService *servicemeshapi.VirtualService
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
				virtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
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
				virtualService: &servicemeshapi.VirtualService{
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
			meshValidator := NewVirtualServiceValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.virtualService)
			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.virtualService, tt.wantErr.reason)
			}
			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}
