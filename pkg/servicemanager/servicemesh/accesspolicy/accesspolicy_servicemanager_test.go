/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package accesspolicy

import (
	"context"
	"errors"
	sdkcommons "github.com/oracle/oci-go-sdk/v65/common"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
)

var (
	opcRetryToken  = "opcRetryToken"
	compartment    = "myCompartment"
	sourceVs       = "my-vs1-id"
	destinationVs  = "my-vs2-id"
	newCompartment = "newCompartment"
	timeNow        = time.Now()
)

func Test_ResolveDependencies(t *testing.T) {
	type fields struct {
		ResolveMeshId                           func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		ResolveVirtualServiceIdAndName          []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		ResolveIngressGatewayIdAndNameAndMeshId []func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
	}
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *conversions.AccessPolicyDependencies
		wantErr error
	}{
		{
			name: "accesspolicy with empty namespace",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name: "my-mesh",
							},
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
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice1-id"),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice2-id"),
						}
						return &virtualServiceR, nil
					},
				},
			},
			want: &conversions.AccessPolicyDependencies{
				MeshId: api.OCID("my-mesh-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice2-id",
						"source":      "my-virtualservice1-id",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "accesspolicy with mesh and virtual services",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice1",
											Namespace: "my-namespace1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice2",
											Namespace: "my-namespace2",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice1-id"),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice2-id"),
						}
						return &virtualServiceR, nil
					},
				},
			},
			want: &conversions.AccessPolicyDependencies{
				MeshId: api.OCID("my-mesh-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice2-id",
						"source":      "my-virtualservice1-id",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "accesspolicy with ingress gateway source",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-ingressgateway1",
											Namespace: "my-namespace1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice2",
											Namespace: "my-namespace2",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice2-id"),
						}
						return &virtualServiceR, nil
					},
				},
				ResolveIngressGatewayIdAndNameAndMeshId: []func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						ingressGatewayR := commons.ResourceRef{
							Id: api.OCID("my-ingressgateway1-id"),
						}
						return &ingressGatewayR, nil
					},
				},
			},
			want: &conversions.AccessPolicyDependencies{
				MeshId: api.OCID("my-mesh-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice2-id",
						"source":      "my-ingressgateway1-id",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "accesspolicy with same mesh and virtual services",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice1",
											Namespace: "my-namespace1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice2",
											Namespace: "my-namespace2",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice1-id"),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice2-id"),
						}
						return &virtualServiceR, nil
					},
				},
			},
			want: &conversions.AccessPolicyDependencies{
				MeshId: api.OCID("my-mesh-id"),
				RefIdForRules: []map[string]api.OCID{
					{
						"destination": "my-virtualservice2-id",
						"source":      "my-virtualservice1-id",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "accesspolicy mesh not found",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("mesh " + string(meshRef.Name) + ":" + meshRef.Namespace + " not found")
				},
				ResolveVirtualServiceIdAndName: nil,
			},
			want:    nil,
			wantErr: errors.New("mesh my-mesh:my-namespace not found"),
		},
		{
			name: "accesspolicy rule source virtual service not found",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice1",
											Namespace: "my-namespace1",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						return nil, errors.New("virtual service " + string(virtualServiceRef.Name) + ":" + virtualServiceRef.Namespace + " not found")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("virtual service my-virtualservice1:my-namespace1 not found"),
		},
		{
			name: "accesspolicy rule destination virtual service not found",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice1",
											Namespace: "my-namespace1",
										},
									},
								},
								Destination: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice2",
											Namespace: "my-namespace2",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-virtualservice1-id"),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						return nil, errors.New("virtual service " + string(virtualServiceRef.Name) + ":" + virtualServiceRef.Namespace + " not found")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("virtual service my-virtualservice2:my-namespace2 not found"),
		},
		{
			name: "accesspolicy mesh is not active",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("mesh is not active yet")
				},
				ResolveVirtualServiceIdAndName: nil,
			},
			want:    nil,
			wantErr: errors.New("mesh is not active yet"),
		},
		{
			name: "accesspolicy source virtual service is not active",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									VirtualService: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-virtualservice1",
											Namespace: "my-namespace1",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						return nil, errors.New("virtual service is not active yet")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("virtual service is not active yet"),
		},
		{
			name: "accesspolicy rule source ingress gateway not found",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-ingressgateway1",
											Namespace: "my-namespace1",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveIngressGatewayIdAndNameAndMeshId: []func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						return nil, errors.New("ingress gateway " + string(ingressGatewayRef.Name) + ":" + ingressGatewayRef.Namespace + " not found")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("ingress gateway my-ingressgateway1:my-namespace1 not found"),
		},
		{
			name: "accesspolicy rule source ingress gateway not active",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      "my-mesh",
								Namespace: "my-namespace",
							},
						},
						Rules: []servicemeshapi.AccessPolicyRule{
							{
								Source: servicemeshapi.TrafficTarget{
									IngressGateway: &servicemeshapi.RefOrId{
										ResourceRef: &servicemeshapi.ResourceRef{
											Name:      "my-ingressgateway1",
											Namespace: "my-namespace1",
										},
									},
								},
							},
						},
					},
				},
			},
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				ResolveIngressGatewayIdAndNameAndMeshId: []func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, ingressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						return nil, errors.New("ingress gateway is not active yet")
					},
				},
			},
			want:    nil,
			wantErr: errors.New("ingress gateway is not active yet"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			resolver := meshMocks.NewMockResolver(ctrl)

			m := &ResourceManager{
				referenceResolver: resolver,
			}
			apDetails := &manager.ResourceDetails{}
			if tt.fields.ResolveMeshId != nil {
				resolver.EXPECT().ResolveMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshId)
			}
			if tt.fields.ResolveVirtualServiceIdAndName != nil {
				for _, ResolveVirtualServiceIdAndName := range tt.fields.ResolveVirtualServiceIdAndName {
					resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(ResolveVirtualServiceIdAndName)
				}
			}
			if tt.fields.ResolveIngressGatewayIdAndNameAndMeshId != nil {
				for _, resolveIngressGatewayId := range tt.fields.ResolveIngressGatewayIdAndNameAndMeshId {
					resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(resolveIngressGatewayId)
				}
			}

			err := m.ResolveDependencies(ctx, tt.args.accessPolicy, apDetails)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, apDetails.ApDetails.Dependencies)
			}
		})
	}
}

func Test_getSDK(t *testing.T) {
	type fields struct {
		GetAccessPolicy func(ctx context.Context, accessPolicyId *api.OCID) (*sdk.AccessPolicy, error)
	}
	tests := []struct {
		name         string
		fields       fields
		accessPolicy *servicemeshapi.AccessPolicy
		wantErr      error
	}{
		{
			name: "valid sdk access policy",
			fields: fields{
				GetAccessPolicy: func(ctx context.Context, accessPolicyId *api.OCID) (*sdk.AccessPolicy, error) {
					return &sdk.AccessPolicy{
						LifecycleState: sdk.AccessPolicyLifecycleStateActive,
					}, nil
				},
			},
			accessPolicy: &servicemeshapi.AccessPolicy{
				Status: servicemeshapi.ServiceMeshStatus{
					AccessPolicyId: "my-accesspolicy",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk access policy not found",
			fields: fields{
				GetAccessPolicy: func(ctx context.Context, accessPolicyId *api.OCID) (*sdk.AccessPolicy, error) {
					return nil, errors.New("access policy not found")
				},
			},
			accessPolicy: &servicemeshapi.AccessPolicy{
				Status: servicemeshapi.ServiceMeshStatus{
					AccessPolicyId: "my-accesspolicy",
				},
			},
			wantErr: errors.New("access policy not found"),
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
			apDetails := &manager.ResourceDetails{}
			if tt.fields.GetAccessPolicy != nil {
				meshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetAccessPolicy)
			}

			err := m.GetResource(ctx, tt.accessPolicy, apDetails)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_updateK8s(t *testing.T) {
	type args struct {
		accessPolicy    *servicemeshapi.AccessPolicy
		sdkAccessPolicy *sdk.AccessPolicy
		oldAccessPolicy *servicemeshapi.AccessPolicy
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		want    *servicemeshapi.AccessPolicy
	}{
		{
			name: "access policy updated and active",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				sdkAccessPolicy: &sdk.AccessPolicy{
					MeshId:         conversions.String("my-mesh-id"),
					Id:             conversions.String("my-accesspolicy-id"),
					LifecycleState: sdk.AccessPolicyLifecycleStateActive,
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.AccessPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AccessPolicy",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-accesspolicy",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:         "my-mesh-id",
					AccessPolicyId: "my-accesspolicy-id",
					RefIdForRules:  make([]map[string]api.OCID, 0),
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
			name: "access policy not active",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:         "my-mesh-id",
						AccessPolicyId: "my-accesspolicy-id",
					},
				},
				sdkAccessPolicy: &sdk.AccessPolicy{
					Id:             conversions.String("my-accesspolicy-id"),
					LifecycleState: sdk.AccessPolicyLifecycleStateFailed,
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.AccessPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AccessPolicy",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-accesspolicy",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:         "my-mesh-id",
					AccessPolicyId: "my-accesspolicy-id",
					RefIdForRules:  make([]map[string]api.OCID, 0),
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
			name: "access policy no update needed",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:         "my-mesh-id",
						AccessPolicyId: "my-accesspolicy-id",
						RefIdForRules:  make([]map[string]api.OCID, 0),
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
				sdkAccessPolicy: &sdk.AccessPolicy{
					Id:             conversions.String("my-accesspolicy-id"),
					LifecycleState: sdk.AccessPolicyLifecycleStateActive,
				},
				oldAccessPolicy: &servicemeshapi.AccessPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-accesspolicy",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-accesspolicy",
				},
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:         "my-mesh-id",
					AccessPolicyId: "my-accesspolicy-id",
					RefIdForRules:  make([]map[string]api.OCID, 0),
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
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.oldAccessPolicy).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AccessPolicy")},
			}
			apDetails := &manager.ResourceDetails{}
			apDetails.ApDetails.SdkAp = tt.args.sdkAccessPolicy
			apDetails.ApDetails.Dependencies = &conversions.AccessPolicyDependencies{
				RefIdForRules: make([]map[string]api.OCID, 0),
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_, err = m.UpdateK8s(ctx, tt.args.accessPolicy, apDetails, false, false)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
				assert.True(t, cmp.Equal(tt.want, tt.args.accessPolicy, opts), "diff", cmp.Diff(tt.want, tt.args.accessPolicy, opts))
			}
		})
	}
}

func Test_Finalize(t *testing.T) {
	m := &ResourceManager{}
	err := m.Finalize(context.Background(), nil)
	assert.NoError(t, err)
}

func Test_Osok_Finalize(t *testing.T) {
	type fields struct {
		DeleteAccessPolicy func(ctx context.Context, accessPolicyId *api.OCID) error
	}
	tests := []struct {
		name         string
		fields       fields
		accessPolicy *servicemeshapi.AccessPolicy
		wantErr      error
	}{
		{
			name: "sdk access policy deleted",
			fields: fields{
				DeleteAccessPolicy: func(ctx context.Context, accessPolicyId *api.OCID) error {
					return nil
				},
			},
			accessPolicy: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					AccessPolicyId: "my-accesspolicy",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk access policy not deleted",
			fields: fields{
				DeleteAccessPolicy: nil,
			},
			accessPolicy: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					AccessPolicyId: "",
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
			f := framework.NewFakeClientFramework(t)
			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				client:            f.K8sClient,
				serviceMeshClient: meshClient,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.DeleteAccessPolicy != nil {
				meshClient.EXPECT().DeleteAccessPolicy(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteAccessPolicy)
			}

			_, err := m.Delete(ctx, tt.accessPolicy)
			assert.True(t, len(tt.accessPolicy.ObjectMeta.Finalizers) != 0)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
		err          error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.AccessPolicy
	}{
		{
			name: "access policy active condition updated with service mesh client error",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access policy active condition updated with service mesh client timeout",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.accessPolicy).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AP")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.accessPolicy, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.accessPolicy, opts), "diff", cmp.Diff(tt.want, tt.args.accessPolicy, opts))
		})
	}
}

func Test_ServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
		err          error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.AccessPolicy
	}{
		{
			name: "access policy dependencies active condition updated with service mesh client error",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access policy dependencies active condition updated with service mesh client timeout",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access policy dependencies active condition updated with empty error message",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access polciy dependencies active condition updated with k8s error message",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-mesh-id is not active yet"),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.DependenciesNotResolved),
								Message:            "my-mesh-id is not active yet",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.accessPolicy).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AP")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.accessPolicy, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.accessPolicy, opts), "diff", cmp.Diff(tt.want, tt.args.accessPolicy, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		accessPolicy *servicemeshapi.AccessPolicy
		err          error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.AccessPolicy
	}{
		{
			name: "access policy configured condition updated with service mesh client error",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access policy configured condition updated with service mesh client timeout",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			name: "access policy configured condition updated with empty error message",
			args: args{
				accessPolicy: &servicemeshapi.AccessPolicy{
					Spec: servicemeshapi.AccessPolicySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.AccessPolicy{
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.accessPolicy).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AP")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.accessPolicy, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.accessPolicy, opts), "diff", cmp.Diff(tt.want, tt.args.accessPolicy, opts))
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	type fields struct {
		ResolveMeshId                  func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		ResolveVirtualServiceIdAndName []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		GetAccessPolicy                func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error)
		GetAccessPolicyNewCompartment  func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error)
		CreateAccessPolicy             func(ctx context.Context, apId *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error)
		UpdateAccessPolicy             func(ctx context.Context, apId *sdk.AccessPolicy) error
		ChangeAccessPolicyCompartment  func(ctx context.Context, apId *api.OCID, compartmentId *api.OCID) error
	}
	tests := []struct {
		name                string
		accessPolicy        *servicemeshapi.AccessPolicy
		fields              fields
		times               int
		wantErr             error
		expectOpcRetryToken bool
	}{
		{
			name:         "AP Create without error",
			accessPolicy: getAccessPolicySpec(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name:         "AP Create with error",
			accessPolicy: getAccessPolicySpec(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error) {
					return nil, errors.New("error in creating accessPolicy")
				},
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in creating accessPolicy"),
		},
		{
			name:         "AP created with error and store retry token",
			accessPolicy: getAccessPolicySpec(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:               1,
			expectOpcRetryToken: true,
			wantErr:             errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name:         "AP created without error and clear retry token",
			accessPolicy: getAccessPolicyWithRetryToken(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy, opcRetryToken *string) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:               1,
			expectOpcRetryToken: false,
			wantErr:             nil,
		},
		{
			name:         "AP Change compartment without error",
			accessPolicy: getAccessPolicyWithDiffCompartmentId(newCompartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: nil,
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy: nil,
				ChangeAccessPolicyCompartment: func(ctx context.Context, apId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name:         "AP Change compartment with error",
			accessPolicy: getAccessPolicyWithDiffCompartmentId(newCompartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: nil,
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy: nil,
				ChangeAccessPolicyCompartment: func(ctx context.Context, apId *api.OCID, compartmentId *api.OCID) error {
					return errors.New("error in changing accessPolicy compartmentId")
				},
			},
			times:   1,
			wantErr: errors.New("error in changing accessPolicy compartmentId"),
		},
		{
			name:         "AP Update without error",
			accessPolicy: getAccessPolicyWithStatus(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: nil,
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy) error {
					return nil
				},
				ChangeAccessPolicyCompartment: nil,
			},
			times:   2,
			wantErr: nil,
		},
		{
			name:         "AP Update with error",
			accessPolicy: getAccessPolicyWithStatus(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: nil,
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				UpdateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy) error {
					return errors.New("error in updating accessPolicy")
				},
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in updating accessPolicy"),
		},
		{
			name:         "Resolve dependencies error on create",
			accessPolicy: getAccessPolicySpec(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("error in resolving dependencies")
				},
				ResolveVirtualServiceIdAndName: nil,
				GetAccessPolicy:                nil,
				CreateAccessPolicy:             nil,
				UpdateAccessPolicy:             nil,
				ChangeAccessPolicyCompartment:  nil,
			},
			times:   1,
			wantErr: errors.New("error in resolving dependencies"),
		},
		{
			name:         "get sdk error",
			accessPolicy: getAccessPolicyWithStatus(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return nil, errors.New("error in getting accessPolicy")
				},
				CreateAccessPolicy:            nil,
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in getting accessPolicy"),
		},
		{
			name:         "sdk accessPolicy is deleted",
			accessPolicy: getAccessPolicyWithStatus(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return &sdk.AccessPolicy{
						LifecycleState: sdk.AccessPolicyLifecycleStateDeleted,
					}, nil
				},
				CreateAccessPolicy:            nil,
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name:         "sdk accessPolicy is failed",
			accessPolicy: getAccessPolicyWithStatus(compartment, sourceVs, destinationVs),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(destinationVs),
						}
						return &virtualServiceR, nil
					},
				},
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return &sdk.AccessPolicy{
						LifecycleState: sdk.AccessPolicyLifecycleStateFailed,
					}, nil
				},
				CreateAccessPolicy:            nil,
				UpdateAccessPolicy:            nil,
				ChangeAccessPolicyCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name:         "AP Update with new compartment",
			accessPolicy: getAccessPolicyWithDiffCompartmentId(newCompartment, sourceVs, "updated-vs-id"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshId := api.OCID("my-mesh-id")
					return &meshId, nil
				},
				ResolveVirtualServiceIdAndName: []func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID(sourceVs),
						}
						return &virtualServiceR, nil
					},
					func(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("updated-vs-id"),
						}
						return &virtualServiceR, nil
					},
				},
				CreateAccessPolicy: nil,
				GetAccessPolicy: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(compartment), nil
				},
				GetAccessPolicyNewCompartment: func(ctx context.Context, apId *api.OCID) (*sdk.AccessPolicy, error) {
					return getSdkAccessPolicy(newCompartment), nil
				},
				UpdateAccessPolicy: func(ctx context.Context, ap *sdk.AccessPolicy) error {
					return nil
				},
				ChangeAccessPolicyCompartment: func(ctx context.Context, apId *api.OCID, compartmentId *api.OCID) error {
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
			f := framework.NewFakeClientFramework(t)
			meshClient := meshMocks.NewMockServiceMeshClient(controller)
			resolver := meshMocks.NewMockResolver(controller)

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("AP")},
				serviceMeshClient: meshClient,
				client:            f.K8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.ResolveMeshId != nil {
				resolver.EXPECT().ResolveMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshId).AnyTimes()
			}

			if tt.fields.ResolveVirtualServiceIdAndName != nil {
				for _, ResolveVirtualServiceIdAndName := range tt.fields.ResolveVirtualServiceIdAndName {
					resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(ResolveVirtualServiceIdAndName).AnyTimes()
				}
			}

			if tt.fields.GetAccessPolicy != nil {
				meshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetAccessPolicy).Times(tt.times)
			}

			if tt.fields.CreateAccessPolicy != nil {
				meshClient.EXPECT().CreateAccessPolicy(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateAccessPolicy).Times(1)
			}

			if tt.fields.UpdateAccessPolicy != nil {
				meshClient.EXPECT().UpdateAccessPolicy(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateAccessPolicy)
			}

			if tt.fields.ChangeAccessPolicyCompartment != nil {
				meshClient.EXPECT().ChangeAccessPolicyCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeAccessPolicyCompartment)
			}

			assert.NoError(t, f.K8sClient.Create(ctx, tt.accessPolicy))

			var err error
			for i := 0; i < tt.times; i++ {
				_, err = m.CreateOrUpdate(ctx, tt.accessPolicy, ctrl.Request{})
			}

			if tt.fields.GetAccessPolicyNewCompartment != nil {
				meshClient.EXPECT().GetAccessPolicy(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetAccessPolicyNewCompartment).Times(1)
				_, err = m.CreateOrUpdate(ctx, tt.accessPolicy, ctrl.Request{})
			}

			key := types.NamespacedName{Name: "my-accessPolicy", Namespace: "my-namespace"}
			curAp := &servicemeshapi.AccessPolicy{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curAp))
			if tt.expectOpcRetryToken {
				assert.NotNil(t, curAp.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curAp.Status.OpcRetryToken)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateOpcRetryToken(t *testing.T) {
	tests := []struct {
		name                  string
		accessPolicy          *servicemeshapi.AccessPolicy
		opcRetryToken         *string
		expectedOpcRetryToken *string
	}{
		{
			name: "add opc token for new request",
			accessPolicy: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-accessPolicy",
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			opcRetryToken:         &opcRetryToken,
			expectedOpcRetryToken: &opcRetryToken,
		},
		{
			name: "delete opc token from status",
			accessPolicy: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-accessPolicy",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					AccessPolicyId: "my-accessPolicy-id",
					MeshId:         "my-mesh-id",
					OpcRetryToken:  &opcRetryToken,
				},
			},
			opcRetryToken:         nil,
			expectedOpcRetryToken: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			serviceMeshClient := meshMocks.NewMockServiceMeshClient(controller)
			f := framework.NewFakeClientFramework(t)

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				serviceMeshClient: serviceMeshClient,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			key := types.NamespacedName{Name: "my-accessPolicy"}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.accessPolicy))
			_ = m.UpdateOpcRetryToken(ctx, tt.accessPolicy, tt.opcRetryToken)
			curAp := &servicemeshapi.AccessPolicy{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curAp))
			assert.Same(t, tt.expectedOpcRetryToken, tt.opcRetryToken)
		})
	}
}

func getAccessPolicySpec(compartment string, sourceVs string, destinationVs string) *servicemeshapi.AccessPolicy {
	return &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-accessPolicy",
		},
		Spec: servicemeshapi.AccessPolicySpec{
			CompartmentId: api.OCID(compartment),
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name: "my-mesh",
				},
			},
			Rules: []servicemeshapi.AccessPolicyRule{
				{
					Action: servicemeshapi.ActionTypeAllow,
					Source: servicemeshapi.TrafficTarget{
						VirtualService: &servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      servicemeshapi.Name(sourceVs),
								Namespace: "my-namespace",
							},
						},
					},
					Destination: servicemeshapi.TrafficTarget{
						VirtualService: &servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Name:      servicemeshapi.Name(destinationVs),
								Namespace: "my-namespace",
							},
						},
					},
				},
			},
		},
	}
}

func getAccessPolicyWithStatus(compartment string, sourceVs string, destinationVs string) *servicemeshapi.AccessPolicy {
	ap := getAccessPolicySpec(compartment, sourceVs, destinationVs)
	ap.Status = servicemeshapi.ServiceMeshStatus{
		AccessPolicyId:  "my-accessPolicy-id",
		MeshId:          "my-mesh-id",
		LastUpdatedTime: &metav1.Time{Time: timeNow},
	}
	return ap
}

func getAccessPolicyWithRetryToken(compartment string, sourceVs string, destinationVs string) *servicemeshapi.AccessPolicy {
	ap := getAccessPolicySpec(compartment, sourceVs, destinationVs)
	ap.Generation = 1
	ap.Status = servicemeshapi.ServiceMeshStatus{
		Conditions: []servicemeshapi.ServiceMeshCondition{
			{
				Type: servicemeshapi.ServiceMeshConfigured,
				ResourceCondition: servicemeshapi.ResourceCondition{
					Status:             metav1.ConditionUnknown,
					Reason:             "Timeout",
					Message:            "Timeout",
					ObservedGeneration: 1,
				},
			},
		},
		OpcRetryToken: &opcRetryToken}
	return ap
}

func getAccessPolicyWithDiffCompartmentId(compartment string, sourceVs string, destinationVs string) *servicemeshapi.AccessPolicy {
	ap := getAccessPolicyWithStatus(compartment, sourceVs, destinationVs)
	ap.Generation = 2
	newCondition := servicemeshapi.ServiceMeshCondition{
		Type: servicemeshapi.ServiceMeshActive,
		ResourceCondition: servicemeshapi.ResourceCondition{
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	ap.Status.Conditions = append(ap.Status.Conditions, newCondition)
	return ap
}

func getSdkAccessPolicy(compartment string) *sdk.AccessPolicy {
	return &sdk.AccessPolicy{
		Id:             conversions.String("my-accessPolicy-id"),
		MeshId:         conversions.String("my-mesh-id"),
		LifecycleState: sdk.AccessPolicyLifecycleStateActive,
		CompartmentId:  conversions.String(compartment),
		TimeCreated:    &sdkcommons.SDKTime{Time: timeNow},
		TimeUpdated:    &sdkcommons.SDKTime{Time: timeNow},
	}
}
