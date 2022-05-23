/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sdkcommons "github.com/oracle/oci-go-sdk/v65/common"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	meshMocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	meshID                 = api.OCID("my-mesh-id")
	certificateAuthorityId = "certificate-authority-id"
	igDescription          = servicemeshapi.Description("This is Ingress Gateway")
	igDescription1         = servicemeshapi.Description("This is Ingress Gateway 1")
	opcRetryToken          = "opcRetryToken"
	timeNow                = time.Now()
)

func Test_GetIg(t *testing.T) {
	type fields struct {
		GetIngressGateway func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error)
	}
	tests := []struct {
		name           string
		fields         fields
		ingressGateway *servicemeshapi.IngressGateway
		wantErr        error
	}{
		{
			name: "valid sdk ingress gateway",
			fields: fields{
				GetIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
					return &sdk.IngressGateway{
						LifecycleState: sdk.IngressGatewayLifecycleStateActive,
						CompartmentId:  conversions.String("ocid1.ig.oc1.iad.1"),
					}, nil
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "my-ingressgateway-id",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk ingress gateway not found",
			fields: fields{
				GetIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) (*sdk.IngressGateway, error) {
					return nil, errors.New("ingress gateway not found")
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "my-ingressgateway-id",
				},
			},
			wantErr: errors.New("ingress gateway not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(controller)

			m := &ResourceManager{
				serviceMeshClient: meshClient,
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
			}

			if tt.fields.GetIngressGateway != nil {
				meshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIngressGateway)
			}

			err := m.GetResource(ctx, tt.ingressGateway, &manager.ResourceDetails{})
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
		ig *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name string
		args args
		want *sdk.IngressGateway
	}{
		{
			name: "Converts API IngressGateway to SDK IngressGateway - base case",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
						Name:      "my-ingressgateway",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Description:   &igDescription,
						CompartmentId: "my-compartment-id",
						Mesh: servicemeshapi.RefOrId{
							ResourceRef: &servicemeshapi.ResourceRef{
								Namespace: "my-namespace",
								Name:      "my-mesh",
							},
						},
						Hosts: []servicemeshapi.IngressGatewayHost{
							{
								Name:      "testHost",
								Hostnames: []string{"test.com"},
								Listeners: []servicemeshapi.IngressGatewayListener{},
							},
						},
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						IngressGatewayId: "my-ingressgateway-id",
						MeshId:           "my-mesh-id",
					},
				},
			},
			want: &sdk.IngressGateway{
				Id:            conversions.String("my-ingressgateway-id"),
				CompartmentId: conversions.String("my-compartment-id"),
				Description:   conversions.String("This is Ingress Gateway"),
				Hosts: []sdk.IngressGatewayHost{
					{
						Name:      conversions.String("testHost"),
						Hostnames: []string{"test.com"},
						Listeners: []sdk.IngressGatewayListener{},
					},
				},
				MeshId:       conversions.String("my-mesh-id"),
				Name:         conversions.String("my-namespace/my-ingressgateway"),
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(controller)

			m := &ResourceManager{
				serviceMeshClient: meshClient,
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
			}
			igDetails := &manager.ResourceDetails{}
			igDetails.IgDetails.MeshId = &meshID
			_ = m.BuildSdk(tt.args.ig, igDetails)
			assert.Equal(t, tt.want, igDetails.IgDetails.BuildSdkIg)
		})
	}
}

func Test_Finalizer(t *testing.T) {
	type env struct {
		igds           []*servicemeshapi.IngressGatewayDeployment
		igrts          []*servicemeshapi.IngressGatewayRouteTable
		accessPolicies []*servicemeshapi.AccessPolicy
	}
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "ingressgateway has no subresources associated",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-ingressgateway",
						Namespace:  "my-namespace",
						Finalizers: []string{commons.IngressGatewayFinalizer},
					},
					Spec: servicemeshapi.IngressGatewaySpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						IngressGatewayId: "my-ingressgateway-id",
						MeshId:           "my-mesh-id",
					},
				},
			},
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{},
			},
			wantErr: nil,
		},
		{
			name: "ingressgateway has ingress gateway deployment associated",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-ingressgateway",
						Namespace:  "my-namespace",
						Finalizers: []string{commons.IngressGatewayFinalizer},
					},
					Spec: servicemeshapi.IngressGatewaySpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						IngressGatewayId: "my-ingressgateway-id",
						MeshId:           "my-mesh-id",
					},
				},
			},
			env: env{
				igds: []*servicemeshapi.IngressGatewayDeployment{getIgd()},
			},
			wantErr: errors.New("cannot delete ingress gateway when there are ingress gateway deployment resources associated"),
		},
		{
			name: "ingressgateway has ingress gateway route table associated",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-ingressgateway",
						Namespace:  "my-namespace",
						Finalizers: []string{commons.IngressGatewayFinalizer},
					},
					Spec: servicemeshapi.IngressGatewaySpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						IngressGatewayId: "my-ingressgateway-id",
						MeshId:           "my-mesh-id",
					},
				},
			},
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{getIgrt()},
			},
			wantErr: errors.New("cannot delete ingress gateway when there are ingress gateway route table resources associated"),
		},
		{
			name: "ingressgateway has access policy associated",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "my-ingressgateway",
						Namespace:  "my-namespace",
						Finalizers: []string{commons.IngressGatewayFinalizer},
					},
					Spec: servicemeshapi.IngressGatewaySpec{},
					Status: servicemeshapi.ServiceMeshStatus{
						IngressGatewayId: "my-ingressgateway-id",
						MeshId:           "my-mesh-id",
					},
				},
			},
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{getAp()},
			},
			wantErr: errors.New("cannot delete ingress gateway when there are access policy resources associated"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testFramework := framework.NewFakeClientFramework(t)
			resourceManager := &ResourceManager{
				log:    loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
				client: testFramework.K8sClient,
			}

			m := manager.NewServiceMeshServiceManager(testFramework.K8sClient, resourceManager.log, resourceManager)
			assert.NoError(t, testFramework.K8sClient.Create(ctx, tt.args.ingressGateway))
			for _, igd := range tt.env.igds {
				err := testFramework.K8sClient.Create(ctx, igd.DeepCopy())
				assert.NoError(t, err)
			}

			for _, igrt := range tt.env.igrts {
				err := testFramework.K8sClient.Create(ctx, igrt.DeepCopy())
				assert.NoError(t, err)
			}

			for _, ap := range tt.env.accessPolicies {
				err := testFramework.K8sClient.Create(ctx, ap.DeepCopy())
				assert.NoError(t, err)
			}

			_, err := m.Delete(ctx, tt.args.ingressGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.True(t, len(tt.args.ingressGateway.ObjectMeta.Finalizers) != 0)
			} else {
				assert.NoError(t, err)
				assert.True(t, len(tt.args.ingressGateway.ObjectMeta.Finalizers) == 0)
			}
		})
	}
}

func Test_Finalize(t *testing.T) {

	type env struct {
		igrts          []*servicemeshapi.IngressGatewayRouteTable
		accessPolicies []*servicemeshapi.AccessPolicy
	}
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		wantErr error
	}{
		{
			name: "ingressgateway has no subresources associated",
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
			},
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{},
			},
			wantErr: nil,
		},
		{
			name: "ingressgateway has ingress gateway route table associated",
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
			},
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{getIgrt()},
			},
			wantErr: errors.New("cannot delete ingress gateway when there are ingress gateway route table resources associated"),
		},
		{
			name: "ingressgateway has ingress gateway access policy associated",
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
			},
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{getAp()},
			},
			wantErr: errors.New("cannot delete ingress gateway when there are access policy resources associated"),
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

			for _, igrt := range tt.env.igrts {
				err := k8sClient.Create(ctx, igrt.DeepCopy())
				assert.NoError(t, err)
			}

			for _, ap := range tt.env.accessPolicies {
				err := k8sClient.Create(ctx, ap.DeepCopy())
				assert.NoError(t, err)
			}

			err := m.Finalize(ctx, tt.args.ingressGateway)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_HasIngressGatewayRouteTables(t *testing.T) {
	type env struct {
		igrts []*servicemeshapi.IngressGatewayRouteTable
	}
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "ingressgateway has ingressGatewayRouteTable dependencies",
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{getIgrt()},
			},
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
			},
			want: true,
		},
		{
			name: "ingressGateway does not have ingressGatewayRouteTable dependencies",
			env: env{
				igrts: []*servicemeshapi.IngressGatewayRouteTable{},
			},
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
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

			for _, igrt := range tt.env.igrts {
				err := k8sClient.Create(ctx, igrt.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasIngressGatewayRouteTables(ctx, tt.args.ingressGateway)
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

func Test_HasAccessPolicies(t *testing.T) {
	type env struct {
		accessPolicies []*servicemeshapi.AccessPolicy
	}
	type args struct {
		ingressGateway *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "ingressgateway has accessPolicy dependencies",
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{getAp()},
			},
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
			},
			want: true,
		},
		{
			name: "ingressGateway does not have accessPolicy dependencies",
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{},
			},
			args: args{
				ingressGateway: getIgWithStatus("myCompartment", "myHost"),
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

			for _, ap := range tt.env.accessPolicies {
				err := k8sClient.Create(ctx, ap.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasAccessPolicies(ctx, tt.args.ingressGateway)
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

func Test_OsokFinalize(t *testing.T) {
	type fields struct {
		DeleteIngressGateway func(ctx context.Context, ingressGatewayId *api.OCID) error
	}
	tests := []struct {
		name           string
		fields         fields
		ingressGateway *servicemeshapi.IngressGateway
		wantErr        error
	}{
		{
			name: "sdk ingress gateway deleted",
			fields: fields{
				DeleteIngressGateway: func(ctx context.Context, ingressGatewayId *api.OCID) error {
					return nil
				},
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "my-ingressgateway-id",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk ingress gateway not deleted",
			fields: fields{
				DeleteIngressGateway: nil,
			},
			ingressGateway: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayId: "",
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

			if tt.fields.DeleteIngressGateway != nil {
				meshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteIngressGateway)
			}

			_, err := m.Delete(ctx, tt.ingressGateway)
			assert.True(t, len(tt.ingressGateway.ObjectMeta.Finalizers) != 0)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_UpdateK8s(t *testing.T) {
	type args struct {
		ingressGateway    *servicemeshapi.IngressGateway
		sdkIngressGateway *sdk.IngressGateway
		oldIngressGateway *servicemeshapi.IngressGateway
	}
	tests := []struct {
		name     string
		args     args
		wantErr  error
		want     *servicemeshapi.IngressGateway
		response *servicemanager.OSOKResponse
	}{
		{
			name: "ingress gateway updated and active",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				sdkIngressGateway: &sdk.IngressGateway{
					MeshId:         conversions.String("my-mesh-id"),
					Id:             conversions.String("my-ingressgateway-id"),
					LifecycleState: sdk.IngressGatewayLifecycleStateActive,
					Mtls: &sdk.IngressGatewayMutualTransportLayerSecurity{
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.IngressGateway{
				TypeMeta: metav1.TypeMeta{
					Kind:       "IngressGateway",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-ingressgateway",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.IngressGatewaySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					IngressGatewayId: "my-ingressgateway-id",
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
					IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
						CertificateId: api.OCID(certificateAuthorityId),
					},
				},
			},
		},
		{
			name: "ingress gateway not active",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
					},
				},
				sdkIngressGateway: &sdk.IngressGateway{
					Id:             conversions.String("my-ingressgateway-id"),
					LifecycleState: sdk.IngressGatewayLifecycleStateFailed,
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.IngressGateway{
				TypeMeta: metav1.TypeMeta{
					Kind:       "IngressGateway",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-ingressgateway",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.IngressGatewaySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					IngressGatewayId: "my-ingressgateway-id",
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
			name: "ingress gateway update with status as unknown ",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionUnknown,
								},
							},
						},
						IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
							CertificateId: api.OCID(certificateAuthorityId),
						},
					},
				},
				sdkIngressGateway: &sdk.IngressGateway{
					Id:             conversions.String("my-ingressgateway-id"),
					LifecycleState: sdk.IngressGatewayLifecycleStateUpdating,
					Mtls: &sdk.IngressGatewayMutualTransportLayerSecurity{
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
				},
			},
			wantErr:  nil,
			response: &servicemanager.OSOKResponse{ShouldRequeue: true},
			want: &servicemeshapi.IngressGateway{
				TypeMeta: metav1.TypeMeta{
					Kind:       "IngressGateway",
					APIVersion: "servicemesh.oci.oracle.com/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "my-ingressgateway",
					ResourceVersion: "1",
				},
				Spec: servicemeshapi.IngressGatewaySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					IngressGatewayId: "my-ingressgateway-id",
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(commons.LifecycleStateChanged),
								Message: string(commons.ResourceUpdating),
							},
						},
					},
				},
			},
		},
		{
			name: "ingress gateway no update needed",
			args: args{
				ingressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId:           "my-mesh-id",
						IngressGatewayId: "my-ingressgateway-id",
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
						IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
							CertificateId: api.OCID(certificateAuthorityId),
						},
					},
				},
				sdkIngressGateway: &sdk.IngressGateway{
					Id:             conversions.String("my-ingressgateway-id"),
					LifecycleState: sdk.IngressGatewayLifecycleStateActive,
					Mtls: &sdk.IngressGatewayMutualTransportLayerSecurity{
						CertificateId: conversions.String(certificateAuthorityId),
					},
				},
				oldIngressGateway: &servicemeshapi.IngressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-ingressgateway",
					},
				},
			},
			wantErr: nil,
			want: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-ingressgateway",
				},
				Spec: servicemeshapi.IngressGatewaySpec{
					Mesh: servicemeshapi.RefOrId{
						Id: "my-mesh-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					IngressGatewayId: "my-ingressgateway-id",
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
					IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
						CertificateId: api.OCID(certificateAuthorityId),
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.oldIngressGateway).Build()

			resourceManager := &ResourceManager{
				log:    loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
				client: k8sClient,
			}
			igDetails := &manager.ResourceDetails{}
			igDetails.IgDetails.SdkIg = tt.args.sdkIngressGateway
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			response, err := m.UpdateK8s(ctx, tt.args.ingressGateway, igDetails, false, false)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				if tt.response != nil {
					assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
				} else {
					opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
					assert.True(t, cmp.Equal(tt.want, tt.args.ingressGateway, opts), "diff", cmp.Diff(tt.want, tt.args.ingressGateway, opts))
				}
			}
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	type fields struct {
		ResolveMeshId       func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
		GetIg               func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error)
		GetIgNewCompartment func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error)
		CreateIg            func(ctx context.Context, ig *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error)
		UpdateIg            func(ctx context.Context, ig *sdk.IngressGateway) error
		ChangeIgCompartment func(ctx context.Context, igId *api.OCID, compartmentId *api.OCID) error
	}
	tests := []struct {
		name                string
		ig                  *servicemeshapi.IngressGateway
		fields              fields
		wantErr             error
		times               int
		expectOpcRetryToken bool
		doNotRequeue        bool
	}{
		{
			name: "ingress gateway created without error",
			ig:   getIgSpec("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: func(ctx context.Context, ig *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				UpdateIg:            nil,
				ChangeIgCompartment: nil,
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "ingress gateway created with error",
			ig:   getIgSpec("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: func(ctx context.Context, ig *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error) {
					return nil, errors.New("error in creating ingress gateway")
				},
				ChangeIgCompartment: nil,
				UpdateIg:            nil,
			},
			times:   1,
			wantErr: errors.New("error in creating ingress gateway"),
		},
		{name: "ingress gateway created with error",
			ig: getIgSpec("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: func(ctx context.Context, ig *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
				ChangeIgCompartment: nil,
				UpdateIg:            nil,
			},
			times:               1,
			expectOpcRetryToken: true,
			wantErr:             errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{name: "ingress gateway created without error and clear retry token",
			ig: getIgWithStatusWithRetryToken("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: func(ctx context.Context, ig *sdk.IngressGateway, opcRetryToken *string) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				ChangeIgCompartment: nil,
				UpdateIg:            nil,
			},
			times:               1,
			expectOpcRetryToken: false,
		},
		{
			name: "ingress gateway changed compartment without error",
			ig:   getIgWithDiffCompartmentId("newCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: nil,
				GetIg: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				UpdateIg: nil,
				ChangeIgCompartment: func(ctx context.Context, igId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "ingress gateway change compartment with error",
			ig:   getIgWithDiffCompartmentId("newCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				CreateIg: nil,
				GetIg: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				UpdateIg: nil,
				ChangeIgCompartment: func(ctx context.Context, igId *api.OCID, compartmentId *api.OCID) error {
					return errors.New("error in changing ingress gateway compartmentId")
				},
			},
			times:   1,
			wantErr: errors.New("error in changing ingress gateway compartmentId"),
		},
		{
			name: "ingress gateway updated without error",
			ig:   getIgWithStatus("myCompartment", "newHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				CreateIg: nil,
				UpdateIg: func(ctx context.Context, ig *sdk.IngressGateway) error {
					return nil
				},
				ChangeIgCompartment: nil,
			},
			times:   2,
			wantErr: nil,
		},
		{
			name: "ingress gateway updated with error",
			ig:   getIgWithStatus("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				CreateIg: nil,
				UpdateIg: func(ctx context.Context, igrt *sdk.IngressGateway) error {
					return errors.New("error in updating ingress gateway")
				},
				ChangeIgCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in updating ingress gateway"),
		},
		{
			name: "Resolve dependencies error on create",
			ig:   getIgSpec("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					return nil, errors.New("error in resolving mesh")
				},
				GetIg:               nil,
				CreateIg:            nil,
				UpdateIg:            nil,
				ChangeIgCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in resolving mesh"),
		},
		{
			name: "get sdk with error",
			ig:   getIgWithStatus("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return nil, errors.New("error in getting SDK ingress gateway")
				},
				CreateIg:            nil,
				UpdateIg:            nil,
				ChangeIgCompartment: nil,
			},
			times:   1,
			wantErr: errors.New("error in getting SDK ingress gateway"),
		},
		{
			name: "sdk ig is deleted",
			ig:   getIgWithStatus("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGateway, error) {
					return &sdk.IngressGateway{
						LifecycleState: sdk.IngressGatewayLifecycleStateDeleted,
					}, nil
				},
				CreateIg:            nil,
				UpdateIg:            nil,
				ChangeIgCompartment: nil,
			},
			times:        1,
			wantErr:      errors.New("ingress gateway in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "sdk ig failed",
			ig:   getIgWithStatus("myCompartment", "myHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGateway, error) {
					return &sdk.IngressGateway{
						LifecycleState: sdk.IngressGatewayLifecycleStateFailed,
					}, nil
				},
				CreateIg:            nil,
				UpdateIg:            nil,
				ChangeIgCompartment: nil,
			},
			times:        1,
			wantErr:      errors.New("ingress gateway in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "ingress gateway updated with compartment change",
			ig:   getIgWithDiffCompartmentId("newCompartment", "newHost"),
			fields: fields{
				ResolveMeshId: func(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
					meshID := api.OCID("my-mesh-id")
					return &meshID, nil
				},
				GetIg: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("myCompartment", "myHost"), nil
				},
				GetIgNewCompartment: func(ctx context.Context, igId *api.OCID) (*sdk.IngressGateway, error) {
					return getSdkIg("newCompartment", "myHost"), nil
				},
				CreateIg: nil,
				UpdateIg: func(ctx context.Context, ig *sdk.IngressGateway) error {
					return nil
				},
				ChangeIgCompartment: func(ctx context.Context, igId *api.OCID, compartmentId *api.OCID) error {
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.ig).Build()

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
				serviceMeshClient: meshClient,
				client:            k8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			if tt.fields.ResolveMeshId != nil {
				resolver.EXPECT().ResolveMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveMeshId).AnyTimes()
			}

			if tt.fields.GetIg != nil {
				meshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIg).Times(tt.times)
			}

			if tt.fields.CreateIg != nil {
				meshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateIg)
			}

			if tt.fields.UpdateIg != nil {
				meshClient.EXPECT().UpdateIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateIg)
			}

			if tt.fields.ChangeIgCompartment != nil {
				meshClient.EXPECT().ChangeIngressGatewayCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeIgCompartment)
			}

			var response servicemanager.OSOKResponse
			for i := 0; i < tt.times; i++ {
				response, err = m.CreateOrUpdate(ctx, tt.ig, ctrl.Request{})
			}

			if tt.fields.GetIgNewCompartment != nil {
				meshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIgNewCompartment).Times(1)
				response, err = m.CreateOrUpdate(ctx, tt.ig, ctrl.Request{})
			}

			key := types.NamespacedName{Name: "my-ingressgateway", Namespace: "my-namespace"}
			curIg := &servicemeshapi.IngressGateway{}
			assert.NoError(t, k8sClient.Get(ctx, key, curIg))
			if tt.expectOpcRetryToken {
				assert.NotNil(t, curIg.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curIg.Status.OpcRetryToken)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, !tt.doNotRequeue, response.ShouldRequeue)
		})
	}
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		ig  *servicemeshapi.IngressGateway
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGateway
	}{
		{
			name: "ingress gateway active condition updated with service mesh client error",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "ingress gateway active condition updated with service mesh client timeout",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.ig).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.ig, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.ig, opts), "diff", cmp.Diff(tt.want, tt.args.ig, opts))
		})
	}
}

func Test_UpdateServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		ig  *servicemeshapi.IngressGateway
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGateway
	}{
		{
			name: "Ingress Gateway dependencies active condition updated with service mesh client error",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "Ingress Gateway dependencies active condition updated with service mesh client timout",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "Ingress Gateway dependencies active condition updated with empty error message",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "Ingress Gateway dependencies active condition updated with k8s error message",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-mesh-id is not active yet"),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.ig).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.ig, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.ig, opts), "diff", cmp.Diff(tt.want, tt.args.ig, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		ig  *servicemeshapi.IngressGateway
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGateway
	}{
		{
			name: "ingress gateway configured condition updated with service mesh client error",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "ingress gateway configured condition updated with service mesh client timeout",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			name: "Ingress Gateway configured condition updated with empty error message",
			args: args{
				ig: &servicemeshapi.IngressGateway{
					Spec: servicemeshapi.IngressGatewaySpec{
						Mesh: servicemeshapi.RefOrId{
							Id: "my-mesh-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.IngressGateway{
				Spec: servicemeshapi.IngressGatewaySpec{
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.ig).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.ig, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.ig, opts), "diff", cmp.Diff(tt.want, tt.args.ig, opts))
		})
	}
}

func TestUpdateOpcRetryToken(t *testing.T) {
	tests := []struct {
		name                  string
		ig                    *servicemeshapi.IngressGateway
		opcRetryToken         *string
		expectedOpcRetryToken *string
	}{
		{
			name: "add opc token for new request",
			ig: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-ingressgateway",
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			opcRetryToken:         &opcRetryToken,
			expectedOpcRetryToken: &opcRetryToken,
		},
		{
			name: "delete opc token from status",
			ig: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-ingressgateway",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:           "my-mesh-id",
					IngressGatewayId: "my-ingressgateway-id",
					OpcRetryToken:    &opcRetryToken,
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

			key := types.NamespacedName{Name: "my-ingressgateway"}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.ig))
			_ = m.UpdateOpcRetryToken(ctx, tt.ig, tt.opcRetryToken)
			curIg := &servicemeshapi.IngressGateway{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curIg))
			assert.Same(t, tt.expectedOpcRetryToken, tt.opcRetryToken)
		})
	}
}

func getIgSpec(compartment string, igHost string) *servicemeshapi.IngressGateway {
	return &servicemeshapi.IngressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-ingressgateway",
		},
		Spec: servicemeshapi.IngressGatewaySpec{
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
			Hosts: []servicemeshapi.IngressGatewayHost{
				{
					Hostnames: []string{igHost},
					Listeners: []servicemeshapi.IngressGatewayListener{},
				},
			},
		},
	}
}

func getIgWithStatus(compartment string, igHost string) *servicemeshapi.IngressGateway {
	ig := getIgSpec(compartment, igHost)
	ig.Status = servicemeshapi.ServiceMeshStatus{
		MeshId:           "my-mesh-id",
		IngressGatewayId: "my-ingressgateway-id",
	}
	return ig
}

func getIgWithStatusWithRetryToken(compartment string, igHost string) *servicemeshapi.IngressGateway {
	ig := getIgSpec(compartment, igHost)
	ig.Generation = 1
	ig.Status = servicemeshapi.ServiceMeshStatus{
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
	return ig
}

func getIgWithDiffCompartmentId(compartment string, igHost string) *servicemeshapi.IngressGateway {
	ig := getIgWithStatus(compartment, igHost)
	ig.Generation = 2
	newCondition := servicemeshapi.ServiceMeshCondition{
		Type: servicemeshapi.ServiceMeshActive,
		ResourceCondition: servicemeshapi.ResourceCondition{
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	ig.Status.Conditions = append(ig.Status.Conditions, newCondition)
	return ig
}

func getSdkIg(compartment string, igHost string) *sdk.IngressGateway {
	return &sdk.IngressGateway{
		Id:             conversions.String("my-ingressgateway-id"),
		MeshId:         conversions.String("my-mesh-id"),
		LifecycleState: sdk.IngressGatewayLifecycleStateActive,
		CompartmentId:  conversions.String(compartment),
		Mtls: &sdk.IngressGatewayMutualTransportLayerSecurity{
			CertificateId: conversions.String(certificateAuthorityId),
		},
		Hosts: []sdk.IngressGatewayHost{
			{
				Hostnames: []string{igHost},
				Listeners: []sdk.IngressGatewayListener{},
			},
		},
		TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
		TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
	}
}

func getIgd() *servicemeshapi.IngressGatewayDeployment {
	return &servicemeshapi.IngressGatewayDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-ingressgatewaydeployment",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			IngressGatewayId: "my-ingressgateway-id",
			Conditions:       nil,
		},
	}
}

func getIgrt() *servicemeshapi.IngressGatewayRouteTable {
	return &servicemeshapi.IngressGatewayRouteTable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-ingressgatewayroutetable",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			IngressGatewayId:           "my-ingressgateway-id",
			IngressGatewayRouteTableId: "my-ingressgatewayroutetable-id1",
			Conditions:                 nil,
		},
	}
}

func getAp() *servicemeshapi.AccessPolicy {
	vsIdForRules := make([]map[string]api.OCID, 1)
	vsIdForRules[0] = make(map[string]api.OCID)
	vsIdForRules[0][commons.Source] = "my-ingressgateway-id"
	vsIdForRules[0][commons.Destination] = ""
	return &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-accessPolicy",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId:        "my-mesh-id",
			RefIdForRules: vsIdForRules,
			Conditions:    nil,
		},
	}
}
