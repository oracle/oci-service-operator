/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package accesspolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	serviceMeshManager "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
)

var (
	apName         = servicemeshapi.Name("my-accesspolicy")
	apName1        = servicemeshapi.Name("my-accesspolicy-1")
	apDescription  = servicemeshapi.Description("This is Access Policy")
	apDescription1 = servicemeshapi.Description("This is Access Policy 1")
)

func Test_AccessPolicyValidateCreate(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
	}
	type expectation struct {
		allowed bool
		reason  string
	}
	type fields struct {
		ResolveResourceRef             func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
		ResolveMeshReference           func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error)
		ResolveVirtualServiceReference func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Spec contains only Mesh OCID",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Spec contains both Mesh Reference and Mesh OCID",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsNotUnique)},
		},
		{
			name: "Spec does not contain Mesh Reference and Mesh OCID",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: nil,
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceIsEmpty)},
		},
		{
			name: "Spec contains traffic targets with virtual service ref in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
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
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
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
			name: "Spec contains traffic targets with virtual service OCID in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										Id: "my-vs-ocid",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										Id: "my-vs-ocid1",
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
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
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
			name: "Spec contains traffic targets with virtual service and ingress gateway OCID in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "my-ig-ocid",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										Id: "my-vs-ocid1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "Spec contains traffic targets with virtual service ref and OCID in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
										Id: "my-vs-ocid",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
										Id: "my-vs-ocid1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec contains traffic targets with virtual service and ingress gateway OCID and ref in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-ig",
										},
										Id: "my-ig-ocid",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsNotUnique)},
		},
		{
			name: "Spec contains traffic targets with ingress gateway OCID and ref missing in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{},
								},
								Destination: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.IngressGatewayReferenceIsEmpty)},
		},
		{
			name: "Spec contains one of the traffic targets with virtual service ref and OCID in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
										Id: "my-vs-ocid",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Spec does not contain traffic targets with virtual service ref and OCID in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Spec does not contain virtual service ref and OCID in one of the traffic targets in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Referred Mesh is deleting",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Referred Mesh is deleting",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
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
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.MeshReferenceNotFound)},
		},
		{
			name: "Spec contains traffic targets with deleting virtual service ref in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
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
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
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
			name: "Spec contains traffic targets with non-exist virtual service ref in accessPolicyRule",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-ocid",
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
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
				ResolveMeshReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
					meshRef := &servicemeshapi.Mesh{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return meshRef, nil
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceNotFound)},
		},
		{
			name: "Spec contains metadata name longer than 190 characters",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Thisisaverylongnametotestthefunctionthereisnopointinhavingthislongofanamebutmaybesometimeswewillhavesuchalongnameandweneedtoensurethatourservicecanhandlethisnamewhenitisoverhundredandninetycharacters",
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.AccessPolicySpec{
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
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					vsRef := &servicemeshapi.VirtualService{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: nil,
						},
					}
					return vsRef, nil
				},
			},
			wantErr: expectation{false, string(commons.MetadataNameLengthExceeded)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewAccessPolicyValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AP")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})

			if tt.fields.ResolveMeshReference != nil {
				resolver.EXPECT().ResolveMeshReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshReference).AnyTimes()
			}

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}

			response := meshServiceValidationManager.ValidateCreateRequest(ctx, tt.args.accessPolicy)

			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.accessPolicy, tt.wantErr.reason)
			}

			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_AccessPolicyValidateUpdate(t *testing.T) {
	type args struct {
		accessPolicy    *servicemeshapi.AccessPolicy
		oldAccessPolicy *servicemeshapi.AccessPolicy
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh-1",
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my.mesh.id",
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Update when spec is not changed",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Update when spec name is not supplied in new AP whereas old AP has a spec name",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &apName,
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
			name: "Update when spec name is supplied in new AP whereas old AP does not have spec name",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &apName1,
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Update when spec name is changed from old AP's spec name",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &apName1,
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Name:          &apName,
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
			name: "Update when spec name is not supplied in both old and new AP",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Description:   &apDescription1,
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Description:   &apDescription,
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
			name: "Update when spec is changed but compartmentId remains the same",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Update when spec contains both virtual service OCID and ref in traffic targets",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
										Id: "my-vs-id",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
										Id: "my-vs-id1",
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Update when spec contains both virtual service OCID and ref in one of the traffic targets",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
										Id: "my-vs-id",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsNotUnique)},
		},
		{
			name: "Update when spec does not contain virtual service OCID and ref in traffic targets",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Update when spec does not contain virtual service OCID and ref in one of the traffic targets",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			wantErr: expectation{false, string(commons.VirtualServiceReferenceIsEmpty)},
		},
		{
			name: "Update when spec does not contain type mismatch",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
								},
								Destination: servicemeshapi.TrafficTarget{},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			wantErr: expectation{false, "access policy target cannot be empty"},
		},
		{
			name: "Update when status is added",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
						CompartmentId: "my.compartmentid",
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
			name: "Spec contains deleting virtual service in access policy",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			name: "Spec contains non-exist virtual service in access policy",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 2},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						CompartmentId: "my.compartmentid",
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name: "my-virtualservice2",
										},
									},
								},
							},
						},
					},
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{Generation: 1},
					Spec: servicemeshapi.AccessPolicySpec{
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
			fields: fields{
				ResolveResourceRef: func(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
					return resourceRef
				},
				ResolveVirtualServiceReference: func(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
					return nil, errors.New("CANNOT FETCH")
				},
			},
			wantErr: expectation{false, string(commons.VirtualServiceReferenceNotFound)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			resolver := meshMocks.NewMockResolver(mockCtrl)
			meshValidator := NewAccessPolicyValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AP")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})

			if tt.fields.ResolveResourceRef != nil {
				resolver.EXPECT().ResolveResourceRef(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveResourceRef).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceReference != nil {
				resolver.EXPECT().ResolveVirtualServiceReference(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveVirtualServiceReference).AnyTimes()
			}
			response := meshServiceValidationManager.ValidateUpdateRequest(ctx, tt.args.accessPolicy, tt.args.oldAccessPolicy)

			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.accessPolicy, tt.wantErr.reason)
			}

			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_AccessPolicyValidateDelete(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
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
				accessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
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
				accessPolicy: &servicemeshapi.AccessPolicy{
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
			meshValidator := NewAccessPolicyValidator(resolver, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AP")})
			meshServiceValidationManager := serviceMeshManager.NewServiceMeshValidationManager(meshValidator, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("VS")})
			response := meshServiceValidationManager.ValidateDeleteRequest(ctx, tt.args.accessPolicy)

			if tt.wantErr.reason != "" {
				tt.wantErr.reason = meshErrors.GetValidationErrorMessage(tt.args.accessPolicy, tt.wantErr.reason)
			}

			assert.Equal(t, tt.wantErr.allowed, response.Allowed)
			assert.Equal(t, tt.wantErr.reason, string(response.Result.Reason))
		})
	}
}

func Test_ValidateTypeTargetMismatch(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
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
			name: "target type match all vs",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
								Destination: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{true, ""},
		},
		{
			name: "source target type mismatch",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									ExternalService: &servicemeshapi.ExternalService{
										HttpExternalService: &servicemeshapi.HttpExternalService{
											Hostnames: []string{"test.com"},
											Ports:     []servicemeshapi.Port{1337},
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "invalid source access policy target. source should be one of: allVirtualServices; virtualService; ingressGateway"},
		},
		{
			name: "destination target type mismatch",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "invalid destination access policy target. destination should be one of: allVirtualServices; virtualService; externalService"},
		},
		{
			name: "target type missing",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
								},
								Destination: servicemeshapi.TrafficTarget{},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "access policy target cannot be empty"},
		},
		{
			name: "more than one target type",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										Id: "igId",
									},
									VirtualService: &servicemeshapi.RefOrId{
										Id: "vsId",
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "access policy target cannot contain more than one type"},
		},
		{
			name: "external service target missing type",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
								Destination: servicemeshapi.TrafficTarget{
									ExternalService: &servicemeshapi.ExternalService{},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "missing external service target"},
		},
		{
			name: "external service target more than one type",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Action: servicemeshapi.ActionTypeAllow,
								Source: servicemeshapi.TrafficTarget{
									AllVirtualServices: &servicemeshapi.AllVirtualServices{},
								},
								Destination: servicemeshapi.TrafficTarget{
									ExternalService: &servicemeshapi.ExternalService{
										TcpExternalService:   &servicemeshapi.TcpExternalService{},
										HttpExternalService:  &servicemeshapi.HttpExternalService{},
										HttpsExternalService: &servicemeshapi.HttpsExternalService{},
									},
								},
							},
						},
					},
				},
			},
			wantErr: expectation{false, "cannot specify more than one external service type"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap := tt.args.accessPolicy
			v := &AccessPolicyValidator{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-validator").WithName("AP")},
			}
			allowed, reason := v.validateAccessPolicyTargets(ap)
			assert.Equal(t, tt.wantErr.allowed, allowed)
			assert.Equal(t, tt.wantErr.reason, reason)
		})
	}
}
