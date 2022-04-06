/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package references

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	customCache "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
)

func Test_ResolveMeshReference(t *testing.T) {
	mesh := &servicemeshapi.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-mesh",
		},
	}

	type env struct {
		meshes []*servicemeshapi.Mesh
	}
	type args struct {
		meshRef *servicemeshapi.ResourceRef
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *servicemeshapi.Mesh
		wantErr error
	}{
		{
			name: "mesh can be resolved",
			env: env{
				meshes: []*servicemeshapi.Mesh{mesh},
			},
			args: args{
				meshRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			want: mesh,
		},
		{
			name: "mesh cannot be resolved",
			env: env{
				meshes: []*servicemeshapi.Mesh{},
			},
			args: args{
				meshRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			wantErr: errors.New("meshs.servicemesh.oci.oracle.com \"my-mesh\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()
			r := NewDefaultResolver(k8sClient, nil, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, k8sClient)

			for _, ms := range tt.env.meshes {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveMeshReference(ctx, tt.args.meshRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func TestResolveMeshId(t *testing.T) {
	crdObj := &metav1.ObjectMeta{
		Namespace: "my-namespace",
		Name:      "my-mesh",
	}
	type args struct {
		GetMesh func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error)
	}
	tests := []struct {
		name    string
		meshRef *servicemeshapi.RefOrId
		mesh    *servicemeshapi.Mesh
		args    args
		want    api.OCID
		wantErr error
	}{
		{
			name: "mesh can be resolved ResourceRef",
			meshRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name: "my-mesh",
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
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
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			want: api.OCID("mesh-id"),
		},
		{
			name: "failed to get mesh ResourceRef",
			meshRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			wantErr: errors.New("meshs.servicemesh.oci.oracle.com \"my-mesh\" not found"),
		},
		{
			name: "failed to verify mesh ResourceRef",
			meshRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
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
			wantErr: errors.New("mesh status condition ServiceMeshActive is not yet satisfied"),
		},
		{
			name: "mesh can be resolved MeshId",
			meshRef: &servicemeshapi.RefOrId{
				Id: "mesh-id",
			},
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						Id:             conversions.String("mesh-id"),
						LifecycleState: sdk.MeshLifecycleStateActive,
					}, nil
				},
			},
			want: api.OCID("mesh-id"),
		},
		{
			name: "failed to get mesh MeshId",
			meshRef: &servicemeshapi.RefOrId{
				Id: "mesh-id",
			},
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return nil, errors.New("Not found")
				},
			},
			wantErr: errors.New("Not found"),
		},
		{
			name: "failed to validate mesh MeshId",
			meshRef: &servicemeshapi.RefOrId{
				Id: "mesh-id",
			},
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						Id:             conversions.String("mesh-id"),
						LifecycleState: sdk.MeshLifecycleStateCreating,
					}, nil
				},
			},
			wantErr: errors.New("mesh is not active yet"),
		},
		{
			name: "mesh cannot be resolved as resourceRef is expected to be deleted soon",
			meshRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "my-namespace",
					Name:              "my-mesh",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
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
			wantErr: errors.New("referenced mesh object with name: my-mesh and namespace: my-namespace is marked for deletion"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, f.K8sClient)

			if tt.mesh != nil {
				err := f.K8sClient.Create(ctx, tt.mesh)
				assert.NoError(t, err)
			}

			if tt.args.GetMesh != nil {
				f.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetMesh)
			}

			got, err := r.ResolveMeshId(ctx, tt.meshRef, crdObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
			}
		})
	}
}

func TestResolveVirtualServiceIdAndName(t *testing.T) {
	crdObj := &metav1.ObjectMeta{
		Namespace: "my-namespace",
		Name:      "my-vs",
	}
	type args struct {
		GetVirtualService func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name    string
		vsRef   *servicemeshapi.RefOrId
		vs      *servicemeshapi.VirtualService
		args    args
		want    commons.ResourceRef
		wantErr error
	}{
		{
			name: "vs can be resolved ResourceRef",
			vsRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
			},
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vs-id",
					MeshId:           "mesh-id",
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
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			want: commons.ResourceRef{
				Id:     api.OCID("vs-id"),
				Name:   servicemeshapi.Name("my-namespace/my-vs"),
				MeshId: api.OCID("mesh-id"),
			},
		},
		{
			name: "failed to get vs ResourceRef",
			vsRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
			},
			wantErr: errors.New("virtualservices.servicemesh.oci.oracle.com \"my-vs\" not found"),
		},
		{
			name: "failed to verify vs ResourceRef",
			vsRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
			},
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vs-id",
					MeshId:           "mesh-id",
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
			wantErr: errors.New("virtual service status condition ServiceMeshActive is not yet satisfied"),
		},
		{
			name: "mesh can be resolved vsId",
			vsRef: &servicemeshapi.RefOrId{
				Id: "vs-id",
			},
			args: args{
				GetVirtualService: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						Id:             conversions.String("vs-id"),
						Name:           conversions.String("vs-name"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.VirtualServiceLifecycleStateActive,
					}, nil
				},
			},
			want: commons.ResourceRef{
				Id:     api.OCID("vs-id"),
				Name:   servicemeshapi.Name("vs-name"),
				MeshId: api.OCID("mesh-id"),
			},
		},
		{
			name: "failed to get vs vsId",
			vsRef: &servicemeshapi.RefOrId{
				Id: "vs-id",
			},
			args: args{
				GetVirtualService: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return nil, errors.New("Not found")
				},
			},
			wantErr: errors.New("Not found"),
		},
		{
			name: "failed to validate vs vsId",
			vsRef: &servicemeshapi.RefOrId{
				Id: "vs-id",
			},
			args: args{
				GetVirtualService: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						Id:             conversions.String("vs-id"),
						Name:           conversions.String("vs-name"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.VirtualServiceLifecycleStateCreating,
					}, nil
				},
			},
			wantErr: errors.New("virtual service is not active yet"),
		},
		{
			name: "vs cannot be resolved as resourceRef is expected to be deleted soon",
			vsRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vs",
				},
			},
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "my-namespace",
					Name:              "my-vs",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
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
			wantErr: errors.New("referenced virtual service object with name: my-vs and namespace: my-namespace is marked for deletion"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, f.K8sClient)

			if tt.vs != nil {
				err := f.K8sClient.Create(ctx, tt.vs)
				assert.NoError(t, err)
			}

			if tt.args.GetVirtualService != nil {
				f.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetVirtualService)
			}

			got, err := r.ResolveVirtualServiceIdAndName(ctx, tt.vsRef, crdObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
			}
		})
	}
}

func TestResolveVirtualServiceReference(t *testing.T) {
	virtualService := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualservice",
		},
	}

	type env struct {
		virtualServices []*servicemeshapi.VirtualService
	}
	type args struct {
		virtualServiceRef *servicemeshapi.ResourceRef
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *servicemeshapi.VirtualService
		wantErr error
	}{
		{
			name: "virtual service can be resolved",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{virtualService},
			},
			args: args{
				virtualServiceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-virtualservice",
				},
			},
			want: virtualService,
		},
		{
			name: "virtual service cannot be resolved",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{},
			},
			args: args{
				virtualServiceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-virtualservice",
				},
			},
			wantErr: errors.New("virtualservices.servicemesh.oci.oracle.com \"my-virtualservice\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()
			r := NewDefaultResolver(k8sClient, nil, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, k8sClient)

			for _, ms := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualServiceReference(ctx, tt.args.virtualServiceRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func TestResolveVirtualDeploymentId(t *testing.T) {
	crdObj := &metav1.ObjectMeta{
		Namespace: "my-namespace",
		Name:      "my-vd",
	}
	type args struct {
		GetVirtualDeployment func(ctx context.Context, vdId *api.OCID) (*sdk.VirtualDeployment, error)
	}
	tests := []struct {
		name    string
		vdRef   *servicemeshapi.RefOrId
		vd      *servicemeshapi.VirtualDeployment
		args    args
		want    api.OCID
		wantErr error
	}{
		{
			name: "vd can be resolved ResourceRef",
			vdRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
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
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			want: api.OCID("vd-id"),
		},
		{
			name: "failed to get vd ResourceRef",
			vdRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
			},
			wantErr: errors.New("virtualdeployments.servicemesh.oci.oracle.com \"my-vd\" not found"),
		},
		{
			name: "failed to verify vd ResourceRef",
			vdRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "vd-id",
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
			wantErr: errors.New("virtual deployment status condition ServiceMeshActive is not yet satisfied"),
		},
		{
			name: "vd can be resolved vdId",
			vdRef: &servicemeshapi.RefOrId{
				Id: "vd-id",
			},
			args: args{
				GetVirtualDeployment: func(ctx context.Context, vdId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						Id:             conversions.String("vd-id"),
						LifecycleState: sdk.VirtualDeploymentLifecycleStateActive,
					}, nil
				},
			},
			want: api.OCID("vd-id"),
		},
		{
			name: "failed to get vd vdId",
			vdRef: &servicemeshapi.RefOrId{
				Id: "vd-id",
			},
			args: args{
				GetVirtualDeployment: func(ctx context.Context, vdId *api.OCID) (*sdk.VirtualDeployment, error) {
					return nil, errors.New("Not found")
				},
			},
			wantErr: errors.New("Not found"),
		},
		{
			name: "failed to validate vs vdId",
			vdRef: &servicemeshapi.RefOrId{
				Id: "vd-id",
			},
			args: args{
				GetVirtualDeployment: func(ctx context.Context, vdId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						Id:             conversions.String("vd-id"),
						LifecycleState: sdk.VirtualDeploymentLifecycleStateCreating,
					}, nil
				},
			},
			wantErr: errors.New("virtual deployment is not active yet"),
		},
		{
			name: "vd cannot be resolved as resourceRef is expected to be deleted soon",
			vdRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-vd",
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "my-namespace",
					Name:              "my-vd",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vs-id",
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
			wantErr: errors.New("referenced virtual deployment object with name: my-vd and namespace: my-namespace is marked for deletion"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, f.K8sClient)

			if tt.vd != nil {
				err := f.K8sClient.Create(ctx, tt.vd)
				assert.NoError(t, err)
			}

			if tt.args.GetVirtualDeployment != nil {
				f.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetVirtualDeployment)
			}

			got, err := r.ResolveVirtualDeploymentId(ctx, tt.vdRef, crdObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
			}
		})
	}
}

func Test_ResolveVirtualDeploymentReference(t *testing.T) {
	virtualDeployment := &servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualdeployment",
		},
	}

	type env struct {
		virtualDeployments []*servicemeshapi.VirtualDeployment
	}
	type args struct {
		virtualDeploymentRef *servicemeshapi.ResourceRef
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    *servicemeshapi.VirtualDeployment
		wantErr error
	}{
		{
			name: "virtual deployment can be resolved",
			env: env{
				virtualDeployments: []*servicemeshapi.VirtualDeployment{virtualDeployment},
			},
			args: args{
				virtualDeploymentRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-virtualdeployment",
				},
			},
			want: virtualDeployment,
		},
		{
			name: "virtual deployment cannot be resolved",
			env: env{
				virtualDeployments: []*servicemeshapi.VirtualDeployment{},
			},
			args: args{
				virtualDeploymentRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-virtualdeployment",
				},
			},
			wantErr: errors.New("virtualdeployments.servicemesh.oci.oracle.com \"my-virtualdeployment\" not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()
			r := NewDefaultResolver(k8sClient, nil, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, k8sClient)

			for _, ms := range tt.env.virtualDeployments {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualDeploymentReference(ctx, tt.args.virtualDeploymentRef)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opt := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opt),
					"diff: %v", cmp.Diff(tt.want, got, opt))
			}
		})
	}
}

func TestResolveIngressGatewayIdAndNameAndMeshId(t *testing.T) {
	crdObj := &metav1.ObjectMeta{
		Namespace: "my-namespace",
		Name:      "my-ingressgatewayroutetable",
	}

	type args struct {
		GetIngressGateway func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error)
	}
	tests := []struct {
		name              string
		ingressGatewayRef *servicemeshapi.RefOrId
		ingressGateway    *servicemeshapi.IngressGateway
		args              args
		want              commons.ResourceRef
		wantErr           error
	}{
		{
			name: "ingress gateway can be resolved ResourceRef",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-ingressgateway",
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-ingressgateway",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "ingressgateway-id",
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
						{
							Type: servicemeshapi.ServiceMeshConfigured,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			want: commons.ResourceRef{
				Id:   api.OCID("ingressgateway-id"),
				Name: servicemeshapi.Name("my-namespace/my-ingressgateway"),
			},
		},
		{
			name: "failed to get ingress gateway ResourceRef",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-ingressgateway",
				},
			},
			wantErr: errors.New("ingressgatewaies.servicemesh.oci.oracle.com \"my-ingressgateway\" not found"),
		},
		{
			name: "failed to verify ingress gateway ResourceRef",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-ingressgateway",
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-ingressgateway",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "ingressgateway-id",
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
			wantErr: errors.New("ingress gateway status condition ServiceMeshActive is not yet satisfied"),
		},
		{
			name: "ingress gateway can be resolved IngressGatewayId",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				Id: "ingressgateway-id",
			},
			args: args{
				GetIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
					return &sdk.IngressGateway{
						Id:             conversions.String("ingressgateway-id"),
						Name:           conversions.String("my-ingressgateway"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.IngressGatewayLifecycleStateActive,
					}, nil
				},
			},
			want: commons.ResourceRef{
				Id:     api.OCID("ingressgateway-id"),
				Name:   servicemeshapi.Name("my-ingressgateway"),
				MeshId: api.OCID("mesh-id"),
			},
		},
		{
			name: "failed to get ingress gateway IngressGatewayId",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				Id: "ingressgateway-id",
			},
			args: args{
				GetIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
					return nil, errors.New("Not found")
				},
			},
			wantErr: errors.New("Not found"),
		},
		{
			name: "failed to validate ingress gateway IngressGatewayId",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				Id: "ingressgateway-id",
			},
			args: args{
				GetIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
					return &sdk.IngressGateway{
						Id:             conversions.String("ingressgateway-id"),
						Name:           conversions.String("my-ingressgateway"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.IngressGatewayLifecycleStateCreating,
					}, nil
				},
			},
			wantErr: errors.New("ingress gateway is not active yet"),
		},
		{
			name: "ig cannot be resolved as resourceRef is expected to be deleted soon",
			ingressGatewayRef: &servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-ig",
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         "my-namespace",
					Name:              "my-ig",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
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
			wantErr: errors.New("referenced ingress gateway object with name: my-ig and namespace: my-namespace is marked for deletion"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Resolver")}, nil, f.K8sClient)

			if tt.ingressGateway != nil {
				err := f.K8sClient.Create(ctx, tt.ingressGateway)
				assert.NoError(t, err)
			}

			if tt.args.GetIngressGateway != nil {
				f.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetIngressGateway)
			}

			got, err := r.ResolveIngressGatewayIdAndNameAndMeshId(ctx, tt.ingressGatewayRef, crdObj)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.want, got)
			}
		})
	}
}

func TestGetMeshRefById(t *testing.T) {
	type args struct {
		GetMesh func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error)
	}
	tests := []struct {
		name    string
		meshId  string
		args    args
		want    *commons.MeshRef
		wantErr error
	}{
		{
			name: "get mesh successfully",
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						Id:             conversions.String("mesh-id"),
						DisplayName:    conversions.String("mesh-name"),
						Mtls:           &sdk.MeshMutualTransportLayerSecurity{Minimum: sdk.MutualTransportLayerSecurityModeDisabled},
						LifecycleState: sdk.MeshLifecycleStateActive,
					}, nil
				},
			},
			meshId: "mesh-id",
			want: &commons.MeshRef{
				Id:          "mesh-id",
				DisplayName: "mesh-name",
				Mtls:        servicemeshapi.MeshMutualTransportLayerSecurity{Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled},
			},
		},
		{
			name: "failed to get mesh",
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return nil, errors.New("Not found")
				},
			},
			meshId:  "mesh-id",
			wantErr: errors.New("Not found"),
		},
		{
			name: "error in conversions",
			args: args{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						Id:             conversions.String("mesh-id"),
						DisplayName:    conversions.String("mesh-name"),
						Mtls:           &sdk.MeshMutualTransportLayerSecurity{Minimum: "stricter"},
						LifecycleState: sdk.MeshLifecycleStateActive,
					}, nil
				},
			},
			meshId:  "mesh-id",
			wantErr: errors.New("unknown MTLS mode type"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			testFramework := framework.NewFakeClientFramework(t)
			customCachesConfig := customCache.CustomCacheConfig{ResyncPeriod: 10 * time.Minute, ClientSet: testFramework.K8sClientset, Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("cache").WithName("Mesh")}}
			meshCacheManager := customCachesConfig.NewSharedCaches()
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("cache").WithName("Mesh")}, meshCacheManager, f.K8sClient)

			if tt.args.GetMesh != nil {
				f.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetMesh)
			}

			got, err := r.ResolveMeshRefById(ctx, conversions.OCID(tt.meshId))
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetVirtualServiceNameById(t *testing.T) {
	type args struct {
		GetVirtualService func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name    string
		vsId    string
		args    args
		want    *commons.ResourceRef
		wantErr error
	}{
		{
			name: "get vsName successfully",
			args: args{
				GetVirtualService: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						Id:             conversions.String("vs-id"),
						Name:           conversions.String("vs-name"),
						MeshId:         conversions.String("mesh-id"),
						LifecycleState: sdk.VirtualServiceLifecycleStateActive,
					}, nil
				},
			},
			vsId: "vs-id",
			want: &commons.ResourceRef{
				Id:     "vs-id",
				Name:   "vs-name",
				MeshId: "mesh-id",
			},
		},
		{
			name: "failed to get vsName",
			args: args{
				GetVirtualService: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return nil, errors.New("Not found")
				},
			},
			vsId:    "vs-id",
			wantErr: errors.New("Not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)
			customCachesConfig := customCache.CustomCacheConfig{ResyncPeriod: 10 * time.Minute, ClientSet: f.K8sClientset, Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("cache").WithName("Mesh")}}
			meshCacheManager := customCachesConfig.NewSharedCaches()
			r := NewDefaultResolver(f.K8sClient, f.MeshClient, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("resolver").WithName("Mesh")}, meshCacheManager, f.K8sClient)

			if tt.args.GetVirtualService != nil {
				f.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.args.GetVirtualService)
			}

			got, err := r.ResolveVirtualServiceRefById(ctx, conversions.OCID(tt.vsId))
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_GetVirtualServiceListByNamespace(t *testing.T) {
	virtualService1 := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualservice-1",
		},
	}
	virtualService2 := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualservice-2",
		},
	}
	virtualService3 := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace-foo",
			Name:      "my-virtualservice-3",
		},
	}
	virtualServiceList := servicemeshapi.VirtualServiceList{Items: []servicemeshapi.VirtualService{*virtualService1, *virtualService2}}

	type env struct {
		virtualServices []*servicemeshapi.VirtualService
	}
	type args struct {
		mesh *servicemeshapi.Mesh
	}
	tests := []struct {
		name string
		env  env
		args args
		want servicemeshapi.VirtualServiceList
	}{
		{
			name: "virtual services present in the namespace",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{virtualService1, virtualService2, virtualService3},
			},
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
				},
			},
			want: virtualServiceList,
		},
		{
			name: "no virtual services in the namespace",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{},
			},
			args: args{
				mesh: &servicemeshapi.Mesh{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-mesh",
						Namespace: "my-namespace",
					},
				},
			},
			want: servicemeshapi.VirtualServiceList{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()
			testFramework := framework.NewFakeClientFramework(t)
			customCachesConfig := customCache.CustomCacheConfig{ResyncPeriod: 10 * time.Minute, ClientSet: testFramework.K8sClientset, Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("cache").WithName("Mesh")}}
			meshCacheManager := customCachesConfig.NewSharedCaches()
			r := NewDefaultResolver(k8sClient, nil, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("resolver").WithName("Mesh")}, meshCacheManager, k8sClient)

			for _, ms := range tt.env.virtualServices {
				err := k8sClient.Create(ctx, ms.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := r.ResolveVirtualServiceListByNamespace(ctx, tt.args.mesh.Namespace)
			assert.NoError(t, err)
			opt := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.want, got, opt),
				"diff: %v", cmp.Diff(tt.want, got, opt))
		})
	}
}
