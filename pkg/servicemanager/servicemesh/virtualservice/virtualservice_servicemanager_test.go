/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualservice

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
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
)

var (
	meshID                 = api.OCID("my-mesh-id")
	certificateAuthorityId = "certificate-authority-id"
	vsDescription          = servicemeshapi.Description("This is Virtual Service")
	opcRetryToken          = "opcRetryToken"
	timeNow                = time.Now()
)

func Test_GetVs(t *testing.T) {
	type fields struct {
		GetVirtualService func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error)
	}
	tests := []struct {
		name           string
		fields         fields
		virtualService *servicemeshapi.VirtualService
		wantErr        error
	}{
		{
			name: "valid sdk virtual service",
			fields: fields{
				GetVirtualService: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						LifecycleState: sdk.VirtualServiceLifecycleStateActive,
						CompartmentId:  conversions.String("ocid1.vsrt.oc1.iad.1"),
						TimeCreated:    &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:    &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "my-virtualservice-id",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk virtual service not found",
			fields: fields{
				GetVirtualService: func(ctx context.Context, virtualServiceId *api.OCID) (*sdk.VirtualService, error) {
					return nil, errors.New("virtual service not found")
				},
			},
			virtualService: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "my-virtualservice-id",
				},
			},
			wantErr: errors.New("virtual service not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			serviceMeshClient := meshMocks.NewMockServiceMeshClient(controller)

			m := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")},
				serviceMeshClient: serviceMeshClient,
			}
			vsDetails := &manager.ResourceDetails{}

			if tt.fields.GetVirtualService != nil {
				serviceMeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVirtualService)
			}

			err := m.GetResource(ctx, tt.virtualService, vsDetails)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_BuildSDK(t *testing.T) {
	type args struct {
		vs *servicemeshapi.VirtualService
	}
	tests := []struct {
		name    string
		args    args
		want    *sdk.VirtualService
		wantErr error
	}{
		{
			name: "Converts API ResourceManager to SDK ResourceManager - base case",
			args: args{
				vs: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
						Name:      "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Description: &vsDescription,
						Hosts:       []string{"myhost"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			want: &sdk.VirtualService{
				Id:            conversions.String("my-virtualservice-id"),
				CompartmentId: conversions.String("my-compartment-id"),
				Hosts:         []string{"myhost"},
				MeshId:        conversions.String("my-mesh-id"),
				Description:   conversions.String("This is Virtual Service"),
				Name:          conversions.String("my-namespace/my-virtualservice"),
				FreeformTags:  map[string]string{},
				DefinedTags:   map[string]map[string]interface{}{},
			},
		},
		{
			name: "Converts API ResourceManager to SDK ResourceManager - error converting mtls",
			args: args{
				vs: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
						Name:      "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						CompartmentId: "my-compartment-id",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []string{"myhost"},
						Mtls:  &servicemeshapi.CreateVirtualServiceMutualTransportLayerSecurity{Mode: "stricter"},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			want:    nil,
			wantErr: errors.New("unknown MTLS mode type"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ResourceManager{}
			vsDetails := &manager.ResourceDetails{}
			vsDetails.VsDetails.MeshId = &meshID
			err := m.BuildSdk(tt.args.vs, vsDetails)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, vsDetails.VsDetails.BuildSdkVs)
		})
	}
}

func Test_VirtualService_Finalize(t *testing.T) {
	type env struct {
		vds   []*servicemeshapi.VirtualDeployment
		vsrts []*servicemeshapi.VirtualServiceRouteTable
		igrts []*servicemeshapi.IngressGatewayRouteTable
		ap    []*servicemeshapi.AccessPolicy
	}
	type args struct {
		virtualService *servicemeshapi.VirtualService
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "virtualservice has no subresources associated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-virtualservice",
						Namespace:  "my-namespace",
						Finalizers: []string{meshCommons.VirtualServiceFinalizer},
					},
					Spec: servicemeshapi.VirtualServiceSpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
				igrts: []*servicemeshapi.IngressGatewayRouteTable{},
				ap:    []*servicemeshapi.AccessPolicy{},
			},
			wantErr: nil,
		},
		{
			name: "virtualservice has virtual service route table associated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-virtualservice",
						Namespace:  "my-namespace",
						Finalizers: []string{meshCommons.VirtualServiceFinalizer},
					},
					Spec: servicemeshapi.VirtualServiceSpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{getVsrt()},
			},
			wantErr: errors.New("cannot delete virtual service when there are virtual service route table resources associated"),
		},
		{
			name: "virtualservice has ingress gateway route table associated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-virtualservice",
						Namespace:  "my-namespace",
						Finalizers: []string{meshCommons.VirtualServiceFinalizer},
					},
					Spec: servicemeshapi.VirtualServiceSpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
				igrts: []*servicemeshapi.IngressGatewayRouteTable{getIgrt()},
			},
			wantErr: errors.New("cannot delete virtual service when there are ingress gateway route table resources associated"),
		},
		{
			name: "virtualservice has access policies associated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-virtualservice",
						Namespace:  "my-namespace",
						Finalizers: []string{meshCommons.VirtualServiceFinalizer},
					},
					Spec: servicemeshapi.VirtualServiceSpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
				igrts: []*servicemeshapi.IngressGatewayRouteTable{},
				ap:    []*servicemeshapi.AccessPolicy{getAp()},
			},
			wantErr: errors.New("cannot delete virtual service when there are access policy resources associated"),
		},
		{
			name: "virtualservice has virtual deployment associated",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-virtualservice",
						Namespace:  "my-namespace",
						Finalizers: []string{meshCommons.VirtualServiceFinalizer},
					},
					Spec: servicemeshapi.VirtualServiceSpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						VirtualServiceId: "my-virtualservice-id",
					},
				},
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{getVd()},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
			},
			wantErr: errors.New("cannot delete virtual service when there are virtual deployment resources associated"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testFramework := framework.NewFakeClientFramework(t)

			resourceManager := &ResourceManager{
				log:    loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				client: testFramework.K8sClient,
			}

			m := manager.NewServiceMeshServiceManager(testFramework.K8sClient, resourceManager.log, resourceManager)
			assert.NoError(t, testFramework.K8sClient.Create(ctx, tt.args.virtualService))
			for _, ap := range tt.env.ap {
				err := testFramework.K8sClient.Create(ctx, ap.DeepCopy())
				assert.NoError(t, err)
			}

			for _, igrt := range tt.env.igrts {
				err := testFramework.K8sClient.Create(ctx, igrt.DeepCopy())
				assert.NoError(t, err)
			}

			for _, vsrt := range tt.env.vsrts {
				err := testFramework.K8sClient.Create(ctx, vsrt.DeepCopy())
				assert.NoError(t, err)
			}

			for _, vd := range tt.env.vds {
				err := testFramework.K8sClient.Create(ctx, vd.DeepCopy())
				assert.NoError(t, err)
			}

			_, err := m.Delete(ctx, tt.args.virtualService)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.True(t, len(tt.args.virtualService.ObjectMeta.Finalizers) != 0)
			} else {
				assert.NoError(t, err)
				assert.True(t, len(tt.args.virtualService.ObjectMeta.Finalizers) == 0)
			}
		})
	}
}

func Test_HasDependentResources(t *testing.T) {

	type env struct {
		vds   []*servicemeshapi.VirtualDeployment
		vsrts []*servicemeshapi.VirtualServiceRouteTable
	}
	type args struct {
		virtualService *servicemeshapi.VirtualService
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "virtualservice has no subresources associated",
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
			},
			wantErr: nil,
		},
		{
			name: "virtualservice has virtual service route table associated",
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{getVsrt()},
			},
			wantErr: errors.New("cannot delete virtual service when there are virtual service route table resources associated"),
		},
		{
			name: "virtualservice has virtual deployment associated",
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			env: env{
				vds:   []*servicemeshapi.VirtualDeployment{getVd()},
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
			},
			wantErr: errors.New("cannot delete virtual service when there are virtual deployment resources associated"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()

			m := &ResourceManager{
				client: k8sClient,
			}

			for _, vsrt := range tt.env.vsrts {
				err := k8sClient.Create(ctx, vsrt.DeepCopy())
				assert.NoError(t, err)
			}

			for _, vd := range tt.env.vds {
				err := m.client.Create(ctx, vd.DeepCopy())
				assert.NoError(t, err)
			}

			err := m.Finalize(ctx, tt.args.virtualService)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_HasVirtualServiceRouteTables(t *testing.T) {
	type env struct {
		vsrts []*servicemeshapi.VirtualServiceRouteTable
	}
	type args struct {
		virtualService *servicemeshapi.VirtualService
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "virtualservice has virtualServiceRouteTable dependencies",
			env: env{
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{getVsrt()},
			},
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			want: true,
		},
		{
			name: "virtualService does not have virtualServiceRouteTable dependencies",
			env: env{
				vsrts: []*servicemeshapi.VirtualServiceRouteTable{},
			},
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()

			m := &ResourceManager{
				client: k8sClient,
			}

			for _, vsrt := range tt.env.vsrts {
				err := k8sClient.Create(ctx, vsrt.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasVirtualServiceRouteTables(ctx, tt.args.virtualService)
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

func Test_HasVirtualDeployments(t *testing.T) {

	type env struct {
		vds []*servicemeshapi.VirtualDeployment
	}
	type args struct {
		virtualService *servicemeshapi.VirtualService
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "virtualService has virtualDeployment dependencies",
			env: env{
				vds: []*servicemeshapi.VirtualDeployment{getVd()},
			},
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			want: true,
		},
		{
			name: "virtualService does not have virtualDeployment dependencies",
			env: env{
				vds: []*servicemeshapi.VirtualDeployment{},
			},
			args: args{
				virtualService: getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			k8sSchema := runtime.NewScheme()
			_ = clientgoscheme.AddToScheme(k8sSchema)
			_ = servicemeshapi.AddToScheme(k8sSchema)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).Build()

			m := &ResourceManager{
				client: k8sClient,
			}

			for _, vd := range tt.env.vds {
				err := m.client.Create(ctx, vd.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasVirtualDeployments(ctx, tt.args.virtualService)
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

func Test_Osok_Finalize(t *testing.T) {
	type fields struct {
		DeleteVirtualService func(ctx context.Context, virtualServiceId *api.OCID) error
	}
	tests := []struct {
		name           string
		fields         fields
		virtualService *servicemeshapi.VirtualService
		wantErr        error
	}{
		{
			name: "sdk virtual service deleted",
			fields: fields{
				DeleteVirtualService: func(ctx context.Context, virtualServiceId *api.OCID) error {
					return nil
				},
			},
			virtualService: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "my-virtualservice-id",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk virtual service not deleted",
			fields: fields{
				DeleteVirtualService: nil,
			},
			virtualService: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "",
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

			if tt.fields.DeleteVirtualService != nil {
				meshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteVirtualService)
			}

			_, err := m.Delete(ctx, tt.virtualService)
			assert.True(t, len(tt.virtualService.ObjectMeta.Finalizers) != 0)
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
		virtualService    *servicemeshapi.VirtualService
		sdkVirtualService *sdk.VirtualService
		oldVirtualService *servicemeshapi.VirtualService
	}
	tests := []struct {
		name     string
		args     args
		wantErr  error
		want     *servicemeshapi.VirtualService
		response *servicemanager.OSOKResponse
	}{
		{
			name: "virtual service updated and active",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				sdkVirtualService: &sdk.VirtualService{
					MeshId:         conversions.String("my-mesh-id"),
					Id:             conversions.String("my-virtualservice-id"),
					LifecycleState: sdk.VirtualServiceLifecycleStateActive,
					Mtls: &sdk.MutualTransportLayerSecurity{
						Mode:          sdk.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualService{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualService",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-virtualservice",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceSpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					VirtualServiceId: "my-virtualservice-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  string(meshCommons.Successful),
								Message: string(meshCommons.ResourceActive),
							},
						},
					},
					VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
						Mode:          servicemeshapi.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.OCID(certificateAuthorityId),
					},
				},
			},
		},
		{
			name: "virtual service not active",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
					},
				},
				sdkVirtualService: &sdk.VirtualService{
					Id:             conversions.String("my-virtualservice-id"),
					LifecycleState: sdk.VirtualServiceLifecycleStateFailed,
					Mtls: &sdk.MutualTransportLayerSecurity{
						Mode:          sdk.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualService{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualService",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-virtualservice",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceSpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					VirtualServiceId: "my-virtualservice-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionFalse,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceFailed),
							},
						},
					},
					VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
						Mode:          servicemeshapi.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.OCID(certificateAuthorityId),
					},
				},
			},
		},
		{
			name: "virtual service update with status as unknown ",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
						VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
							Mode:          servicemeshapi.MutualTransportLayerSecurityModeStrict,
							CertificateId: conversions.OCID(certificateAuthorityId),
						},
					},
				},
				sdkVirtualService: &sdk.VirtualService{
					Id:             conversions.String("my-virtualservice-id"),
					LifecycleState: sdk.VirtualServiceLifecycleStateUpdating,
					Mtls: &sdk.MutualTransportLayerSecurity{
						Mode:          sdk.MutualTransportLayerSecurityModeStrict,
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
			want: &servicemeshapi.VirtualService{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualService",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-virtualservice",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceSpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					VirtualServiceId: "my-virtualservice-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceUpdating),
							},
						},
					},
					VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
						Mode:          servicemeshapi.MutualTransportLayerSecurityModeStrict,
						CertificateId: conversions.OCID(certificateAuthorityId),
					},
				},
			},
		},
		{
			name: "virtual service with mtls updated from parent mesh",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						VirtualServiceId: "my-virtualservice-id",
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status:  metav1.ConditionTrue,
									Reason:  string(meshCommons.Successful),
									Message: string(meshCommons.ResourceActive),
								},
							},
						},
					},
				},
				sdkVirtualService: &sdk.VirtualService{
					Id:             conversions.String("my-virtualservice-id"),
					LifecycleState: sdk.VirtualServiceLifecycleStateActive,
					Mtls: &sdk.MutualTransportLayerSecurity{
						Mode:          sdk.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldVirtualService: &servicemeshapi.VirtualService{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-virtualservice",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.VirtualService{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualService",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-virtualservice",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.VirtualServiceSpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					VirtualServiceId: "my-virtualservice-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  string(meshCommons.Successful),
								Message: string(meshCommons.ResourceActive),
							},
						},
					},
					VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
						Mode:          servicemeshapi.MutualTransportLayerSecurityModePermissive,
						CertificateId: conversions.OCID(certificateAuthorityId),
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.oldVirtualService).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			vsDetails := &manager.ResourceDetails{}
			vsDetails.VsDetails.SdkVs = tt.args.sdkVirtualService
			response, err := m.UpdateK8s(ctx, tt.args.virtualService, vsDetails, false, false)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				if tt.response != nil {
					assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
				} else {
					opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
					assert.True(t, cmp.Equal(tt.want, tt.args.virtualService, opts), "diff", cmp.Diff(tt.want, tt.args.virtualService, opts))
				}
			}
		})
	}
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		virtualService *servicemeshapi.VirtualService
		err            error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualService
	}{
		{
			name: "virtual service active condition updated with service mesh client error",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "virtual service active condition updated with service mesh client timeout",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.virtualService).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.virtualService, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.virtualService, opts), "diff", cmp.Diff(tt.want, tt.args.virtualService, opts))
		})
	}
}

func Test_ServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		virtualService *servicemeshapi.VirtualService
		err            error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualService
	}{
		{
			name: "virtual service dependencies active condition updated with service mesh client error",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "virtual service dependencies active condition updated with service mesh client timeout",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "virtual service dependencies active condition updated with empty error message",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
								Reason:             string(meshCommons.Successful),
								Message:            string(meshCommons.DependenciesResolved),
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "virtual service dependencies active condition updated with k8s error message",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-mesh-id is not active yet"),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
								Reason:             string(meshCommons.DependenciesNotResolved),
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.virtualService).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.virtualService, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.virtualService, opts), "diff", cmp.Diff(tt.want, tt.args.virtualService, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		virtualService *servicemeshapi.VirtualService
		err            error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualService
	}{
		{
			name: "virtual service configured condition updated with service mesh client error",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "virtual service configured condition updated with service mesh client timeout",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
			name: "virtual service configured condition updated with empty error message",
			args: args{
				virtualService: &servicemeshapi.VirtualService{
					Spec: servicemeshapi.VirtualServiceSpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualService{
				Spec: servicemeshapi.VirtualServiceSpec{
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
								Reason:             string(meshCommons.Successful),
								Message:            string(meshCommons.ResourceConfigured),
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.virtualService).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.virtualService, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.virtualService, opts), "diff", cmp.Diff(tt.want, tt.args.virtualService, opts))
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	type fields struct {
		ResolveMeshId       func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		GetVs               func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error)
		GetVsNewCompartment func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error)
		CreateVs            func(ctx context.Context, vs *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error)
		UpdateVs            func(ctx context.Context, vs *sdk.VirtualService) error
		ChangeVsCompartment func(ctx context.Context, vsId *api.OCID, compartmentId *api.OCID) error
	}
	tests := []struct {
		name                string
		vs                  *servicemeshapi.VirtualService
		fields              fields
		wantErr             error
		times               int
		expectOpcRetryToken bool
	}{
		{
			name: "virtual service create without error",
			vs:   getVsSpec("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: func(ctx context.Context, vsrt *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "virtual service create with error",
			vs:   getVsSpec("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: func(ctx context.Context, vs *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error) {
					return nil, errors.New("error in creating virtual service")
				},
				ChangeVsCompartment: nil,
				UpdateVs:            nil,
			},
			times:   1,
			wantErr: errors.New("error in creating virtual service"),
		},
		{
			name: "virtual service create with error and store the retry token",
			vs:   getVsSpec("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: func(ctx context.Context, vs *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
				ChangeVsCompartment: nil,
				UpdateVs:            nil,
			},
			times:               1,
			expectOpcRetryToken: true,
			wantErr:             errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name: "virtual service created without error and clear retry token",
			vs:   getVsWithRetryToken("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: func(ctx context.Context, vsrt *sdk.VirtualService, opcRetryToken *string) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				ChangeVsCompartment: nil,
				UpdateVs:            nil,
			},
			times:               1,
			expectOpcRetryToken: false,
		},
		{
			name: "virtual service change compartment without error",
			vs:   getVsWithDiffCompartmentId("newCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: nil,
				GetVs: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				UpdateVs: nil,
				ChangeVsCompartment: func(ctx context.Context, vsId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "virtual service change compartment with error",
			vs:   getVsWithDiffCompartmentId("newCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateVs: nil,
				GetVs: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				UpdateVs: nil,
				ChangeVsCompartment: func(ctx context.Context, vsId *api.OCID, compartmentId *api.OCID) error {
					return errors.New("error in changing virtual service compartmentId")
				},
			},
			times:   1,
			wantErr: errors.New("error in changing virtual service compartmentId"),
		},
		{
			name: "virtual service update without error",
			vs:   getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyDeny),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				CreateVs: nil,
				UpdateVs: func(ctx context.Context, vs *sdk.VirtualService) error {
					return nil
				},
				ChangeVsCompartment: nil,
			},
			times:   2,
			wantErr: nil,
		},
		{
			name: "virtual service update with error",
			vs:   getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyDeny),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				CreateVs: nil,
				UpdateVs: func(ctx context.Context, vsrt *sdk.VirtualService) error {
					return errors.New("error in updating virtual service")
				},
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in updating virtual service"),
		},
		{
			name: "Resolve dependencies error on create",
			vs:   getVsSpec("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("error in resolving mesh")
				},
				GetVs:               nil,
				CreateVs:            nil,
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in resolving mesh"),
		},
		{
			name: "Resolve dependencies error on create with reference is expected to be deleted soon",
			vs:   getVsSpec("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("referenced mesh object with name: my-mesh and namespace: my-namespace is expected to be deleted")
				},
				GetVs:               nil,
				CreateVs:            nil,
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("referenced mesh object with name: my-mesh and namespace: my-namespace is expected to be deleted"),
		},
		{
			name: "get sdk error",
			vs:   getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return nil, errors.New("error in getting SDK virtual service")
				},
				CreateVs:            nil,
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in getting SDK virtual service"),
		},
		{
			name: "sdk vs is deleted",
			vs:   getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						LifecycleState: sdk.VirtualServiceLifecycleStateDeleted,
					}, nil
				},
				CreateVs:            nil,
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "sdk vs is failed",
			vs:   getVsWithStatus("myCompartment", servicemeshapi.RoutingPolicyUniform),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsrtId *api.OCID) (*sdk.VirtualService, error) {
					return &sdk.VirtualService{
						LifecycleState: sdk.VirtualServiceLifecycleStateFailed,
					}, nil
				},
				CreateVs:            nil,
				UpdateVs:            nil,
				ChangeVsCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "virtual service change compartment with other updates",
			vs:   getVsWithDiffCompartmentId("newCompartment", servicemeshapi.RoutingPolicyDeny),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetVs: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("myCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				GetVsNewCompartment: func(ctx context.Context, vsId *api.OCID) (*sdk.VirtualService, error) {
					return getSdkVs("newCompartment", sdk.DefaultVirtualServiceRoutingPolicyTypeUniform), nil
				},
				UpdateVs: func(ctx context.Context, vs *sdk.VirtualService) error {
					return nil
				},
				ChangeVsCompartment: func(ctx context.Context, vsId *api.OCID, compartmentId *api.OCID) error {
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
			meshClient := meshMocks.NewMockServiceMeshClient(controller)
			resolver := meshMocks.NewMockResolver(controller)

			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.vs).Build()

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VirtualService")},
				serviceMeshClient: meshClient,
				client:            k8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			if tt.fields.ResolveMeshId != nil {
				resolver.EXPECT().ResolveMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshId).AnyTimes()
			}

			if tt.fields.GetVs != nil {
				meshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVs).Times(tt.times)
			}

			if tt.fields.CreateVs != nil {
				meshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateVs)
			}

			if tt.fields.UpdateVs != nil {
				meshClient.EXPECT().UpdateVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateVs)
			}

			if tt.fields.ChangeVsCompartment != nil {
				meshClient.EXPECT().ChangeVirtualServiceCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeVsCompartment)
			}

			for i := 0; i < tt.times; i++ {
				_, err = m.CreateOrUpdate(ctx, tt.vs, ctrl.Request{})
			}

			if tt.fields.GetVsNewCompartment != nil {
				meshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetVsNewCompartment).Times(1)
				_, err = m.CreateOrUpdate(ctx, tt.vs, ctrl.Request{})
			}

			key := types.NamespacedName{Name: "my-virtualservice", Namespace: "my-namespace"}
			curVs := &servicemeshapi.VirtualService{}
			assert.NoError(t, k8sClient.Get(ctx, key, curVs))

			if tt.expectOpcRetryToken {
				assert.NotNil(t, curVs.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curVs.Status.OpcRetryToken)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func getVsSpec(compartment string, routingPolicy servicemeshapi.RoutingPolicy) *servicemeshapi.VirtualService {
	return &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualservice",
		},
		Spec: servicemeshapi.VirtualServiceSpec{
			TagResources: api.TagResources{
				FreeFormTags: map[string]string{},
				DefinedTags:  map[string]api.MapValue{},
			},
			CompartmentId: api.OCID(compartment),
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Namespace: "my-namespace",
					Name:      "my-mesh",
				},
			},
			DefaultRoutingPolicy: &servicemeshapi.DefaultRoutingPolicy{
				Type: routingPolicy,
			},
			Hosts: []string{"myhost"},
		},
	}
}

func getVsWithStatus(compartment string, routingPolicy servicemeshapi.RoutingPolicy) *servicemeshapi.VirtualService {
	vs := getVsSpec(compartment, routingPolicy)
	vs.Status = servicemeshapi.ServiceMeshStatus{
		MeshId:           "my-mesh-id",
		VirtualServiceId: "my-virtualservice-id",
	}
	return vs
}

func getVsWithRetryToken(compartment string, routingPolicy servicemeshapi.RoutingPolicy) *servicemeshapi.VirtualService {
	vs := getVsSpec(compartment, routingPolicy)
	vs.Status = servicemeshapi.ServiceMeshStatus{
		OpcRetryToken: &opcRetryToken,
	}
	return vs
}

func getVsWithDiffCompartmentId(compartment string, routingPolicy servicemeshapi.RoutingPolicy) *servicemeshapi.VirtualService {
	vs := getVsWithStatus(compartment, routingPolicy)
	vs.Generation = 2
	newCondition := servicemeshapi.ServiceMeshCondition{
		Type: servicemeshapi.ServiceMeshActive,
		ResourceCondition: servicemeshapi.ResourceCondition{
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	vs.Status.Conditions = append(vs.Status.Conditions, newCondition)
	return vs
}

func getSdkVs(compartment string, routingPolicy sdk.DefaultVirtualServiceRoutingPolicyTypeEnum) *sdk.VirtualService {
	return &sdk.VirtualService{
		Id:             conversions.String("my-virtualservice-id"),
		MeshId:         conversions.String("my-mesh-id"),
		LifecycleState: sdk.VirtualServiceLifecycleStateActive,
		CompartmentId:  conversions.String(compartment),
		Mtls: &sdk.MutualTransportLayerSecurity{
			Mode:          sdk.MutualTransportLayerSecurityModePermissive,
			CertificateId: conversions.String(certificateAuthorityId),
		},
		DefaultRoutingPolicy: &sdk.DefaultVirtualServiceRoutingPolicy{
			Type: routingPolicy,
		},
		TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
		TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
	}
}

func getVsrt() *servicemeshapi.VirtualServiceRouteTable {
	return &servicemeshapi.VirtualServiceRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualserviceroutetable",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			VirtualServiceId:           "my-virtualservice-id",
			VirtualServiceRouteTableId: "my-virtualserviceroutetable-id1",
			Conditions:                 nil,
		},
	}
}

func getIgrt() *servicemeshapi.IngressGatewayRouteTable {
	vsIds := make([][]api.OCID, 1)
	vsIds[0] = make([]api.OCID, 1)
	vsIds[0][0] = "my-virtualservice-id"
	return &servicemeshapi.IngressGatewayRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-ingressGatewayRoutetable",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			VirtualServiceIdForRules: vsIds,
			Conditions:               nil,
		},
	}
}

func getAp() *servicemeshapi.AccessPolicy {
	vsIdForRules := make([]map[string]api.OCID, 1)
	vsIdForRules[0] = make(map[string]api.OCID)
	vsIdForRules[0][meshCommons.Source] = "my-virtualservice-id"
	vsIdForRules[0][meshCommons.Destination] = ""
	return &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-accessPolicy",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			RefIdForRules: vsIdForRules,
			Conditions:    nil,
		},
	}
}

func getVd() *servicemeshapi.VirtualDeployment {
	return &servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-virtualdeployment",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			VirtualServiceId:    "my-virtualservice-id",
			VirtualDeploymentId: "my-virtualdeployment-id1",
			Conditions:          nil,
		},
	}
}
