/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mesh

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
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	meshConversions "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
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
	meshName               = "mesh"
	meshId                 = "mesh-id"
	compartmentId          = "compartment-id"
	compartmentId1         = "compartment-id1"
	certificateAuthorityId = "certificate-authority-id"
	meshDescription        = servicemeshapi.Description("This is Mesh")
	meshDescription1       = servicemeshapi.Description("This is Mesh 1")
	opcRetryToken          = "opcRetryToken"
	timeNow                = time.Now()
)

func TestBuildSDK(t *testing.T) {
	type args struct {
		vs *servicemeshapi.Mesh
	}
	tests := []struct {
		name    string
		args    args
		want    *sdk.Mesh
		wantErr error
	}{
		{
			name: "converts API Mesh to SDK Mesh",
			args: args{
				vs: &servicemeshapi.Mesh{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      meshName,
						Namespace: "my-namespace",
					},
					Spec: servicemeshapi.MeshSpec{
						Description:   &meshDescription,
						CompartmentId: api.OCID(compartmentId),
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID(meshId),
					},
				},
			},
			want: &sdk.Mesh{
				Id:            meshConversions.String(meshId),
				CompartmentId: meshConversions.String(compartmentId),
				Description:   meshConversions.String("This is Mesh"),
				CertificateAuthorities: []sdk.CertificateAuthority{
					{
						Id: meshConversions.String(certificateAuthorityId),
					},
				},
				Mtls: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
				DisplayName:  meshConversions.String("my-namespace/" + meshName),
				FreeformTags: map[string]string{},
				DefinedTags:  map[string]map[string]interface{}{},
			},
		},
		{
			name: "converts API Mesh to SDK Mesh - error converting mtls",
			args: args{
				vs: &servicemeshapi.Mesh{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: meshName,
					},
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: api.OCID(compartmentId),
						TagResources: api.TagResources{
							FreeFormTags: map[string]string{},
							DefinedTags:  map[string]api.MapValue{},
						},
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
						Mtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
							Minimum: "stricter",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{
						MeshId: api.OCID(meshId),
					},
				},
			},
			want:    nil,
			wantErr: errors.New("unknown MTLS mode type"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ResourceManager{}
			meshDetails := &manager.ResourceDetails{}
			err := m.BuildSdk(tt.args.vs, meshDetails)
			if err != nil {
				assert.EqualError(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, meshDetails.MeshDetails.BuildSdkMesh)
		})
	}
}

func TestOsokFinalize(t *testing.T) {
	type fields struct {
		DeleteMesh func(ctx context.Context, meshId *api.OCID) error
	}
	tests := []struct {
		name    string
		fields  fields
		mesh    *servicemeshapi.Mesh
		wantErr error
	}{
		{
			name: "sdk mesh deleted",
			fields: fields{
				DeleteMesh: func(ctx context.Context, meshId *api.OCID) error {
					return nil
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk mesh id is empty",
			fields: fields{
				DeleteMesh: nil,
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "",
				},
			},
			wantErr: nil,
		},
		{
			name: "sdk mesh deletion error",
			fields: fields{
				DeleteMesh: func(ctx context.Context, meshId *api.OCID) error {
					return errors.New("mesh not deleted")
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
				},
			},
			wantErr: errors.New("mesh not deleted"),
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
				client:            f.K8sClient,
				serviceMeshClient: serviceMeshClient,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.DeleteMesh != nil {
				serviceMeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteMesh)
			}

			_, err := m.Delete(ctx, tt.mesh)
			assert.True(t, len(tt.mesh.ObjectMeta.Finalizers) != 0)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMesh(t *testing.T) {
	type fields struct {
		GetMesh func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error)
	}
	tests := []struct {
		name   string
		fields fields
		mesh   *servicemeshapi.Mesh
	}{
		{
			name: "sdk mesh created",
			fields: fields{
				GetMesh: func(ctx context.Context, meshId *api.OCID) (*sdk.Mesh, error) {
					return nil, nil
				},
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
				},
			},
		},
		{
			name: "sdk mesh not created",
			fields: fields{
				GetMesh: nil,
			},
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			serviceMeshClient := meshMocks.NewMockServiceMeshClient(controller)
			m := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				serviceMeshClient: serviceMeshClient,
			}
			meshDetails := &manager.ResourceDetails{}
			if tt.fields.GetMesh != nil {
				serviceMeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetMesh)
			}

			err := m.GetResource(ctx, tt.mesh, meshDetails)
			assert.NoError(t, err)
		})
	}
}

func TestUpdateServiceMeshCondition(t *testing.T) {
	tests := []struct {
		name              string
		mesh              *servicemeshapi.Mesh
		status            metav1.ConditionStatus
		expectedCondition []servicemeshapi.ServiceMeshCondition
		reason            string
		message           string
	}{
		{
			name: "api mesh has no condition and failed to get sdk mesh",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			status: metav1.ConditionUnknown,
			expectedCondition: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             string(meshCommons.DependenciesNotResolved),
						Message:            "failed to get mesh",
						ObservedGeneration: 1,
					},
				},
			},
			reason:  string(meshCommons.DependenciesNotResolved),
			message: "failed to get mesh",
		},
		{
			name: "api mesh has no condition and failed to update sdk mesh",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
				},
			},
			status: metav1.ConditionUnknown,
			expectedCondition: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             string(meshCommons.DependenciesNotResolved),
						Message:            "failed to update mesh",
						ObservedGeneration: 1,
					},
				},
			},
			reason:  string(meshCommons.DependenciesNotResolved),
			message: "failed to update mesh",
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

			key := types.NamespacedName{Name: meshName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.mesh))
			_ = m.UpdateServiceMeshCondition(ctx, tt.mesh, tt.status, tt.reason, tt.message, servicemeshapi.ServiceMeshActive)
			curMesh := &servicemeshapi.Mesh{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curMesh))
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.expectedCondition, curMesh.Status.Conditions, opts))
		})
	}
}

func TestUpdateK8s(t *testing.T) {
	tests := []struct {
		name           string
		mesh           *servicemeshapi.Mesh
		sdkMesh        *sdk.Mesh
		expectedStatus servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "update mesh id and condition the first time",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			sdkMesh: &sdk.Mesh{
				DisplayName:    &meshName,
				Id:             &meshId,
				LifecycleState: sdk.MeshLifecycleStateCreating,
				Mtls: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionUnknown,
							Reason:  string(meshCommons.LifecycleStateChanged),
							Message: string(meshCommons.ResourceCreating),
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedErr: errors.New(meshCommons.UnknownStatus),
		},
		{
			name: "api mesh is created",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceCreating),
							},
						},
					},
				},
			},
			sdkMesh: &sdk.Mesh{
				DisplayName:    &meshName,
				Id:             &meshId,
				LifecycleState: sdk.MeshLifecycleStateActive,
				Mtls: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
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
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedErr: nil,
		},
		{
			name: "sdk mesh is failed",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceCreating),
							},
						},
					},
				},
			},
			sdkMesh: &sdk.Mesh{
				DisplayName:    &meshName,
				Id:             &meshId,
				LifecycleState: sdk.MeshLifecycleStateFailed,
				Mtls: &sdk.MeshMutualTransportLayerSecurity{
					Minimum: sdk.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
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
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
				},
			},
			expectedErr: nil,
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
			meshDetails := &manager.ResourceDetails{}
			meshDetails.MeshDetails.SdkMesh = tt.sdkMesh
			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)
			key := types.NamespacedName{Name: meshName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.mesh))
			_, _ = m.UpdateK8s(ctx, tt.mesh, meshDetails, false, false)
			curMesh := &servicemeshapi.Mesh{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curMesh))
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.expectedStatus, curMesh.Status, opts))
		})
	}
}

func TestCreateOrUpdateMesh(t *testing.T) {
	type fields struct {
		GetMesh               func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error)
		GetMeshNewCompartment func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error)
		CreateMesh            func(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error)
		UpdateMesh            func(ctx context.Context, mesh *sdk.Mesh) error
		DeleteMesh            func(ctx context.Context, id *api.OCID) error
		ChangeMeshCompartment func(ctx context.Context, meshId *api.OCID, compartmentId *api.OCID) error
	}
	tests := []struct {
		name                string
		mesh                *servicemeshapi.Mesh
		fields              fields
		returnedSdkMesh     *sdk.Mesh
		expectedStatus      servicemeshapi.ServiceMeshStatus
		expectedErr         error
		response            *servicemanager.OSOKResponse
		expectOpcRetryToken bool
		doNotRequeue        bool
	}{
		{
			name: "sdk mesh not created",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				CreateMesh: func(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateCreating,
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModePermissive,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionUnknown,
							Reason:  string(meshCommons.LifecycleStateChanged),
							Message: string(meshCommons.ResourceCreating),
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModePermissive,
				},
			},
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
			expectedErr: nil,
		},
		{
			name: "sdk mesh get error",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceCreating),
							},
						},
					},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return nil, errors.New("Not found")
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionUnknown,
							Reason:  "NotFound",
							Message: "Not found",
						},
					},
				},
			},
			expectedErr: errors.New("Not found"),
		},
		{
			name: "sdk mesh is created",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(meshCommons.LifecycleStateChanged),
								Message: string(meshCommons.ResourceCreating),
							},
						},
					},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateActive,
						CompartmentId:  &compartmentId,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionTrue,
							Reason:  string(meshCommons.Successful),
							Message: string(meshCommons.ResourceActive),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							ObservedGeneration: 1,
							Status:             metav1.ConditionTrue,
							Reason:             string(meshCommons.Successful),
							Message:            string(meshCommons.DependenciesResolved),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							ObservedGeneration: 1,
							Status:             metav1.ConditionTrue,
							Reason:             string(meshCommons.Successful),
							Message:            string(meshCommons.ResourceConfigured),
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
				},
				LastUpdatedTime: &metav1.Time{Time: timeNow},
			},
			expectedErr: nil,
		},
		{
			name: "sdk mesh is deleted",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
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
					MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
						Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
					},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateDeleted,
						CompartmentId:  &compartmentId,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionFalse,
							Reason:  string(meshCommons.LifecycleStateChanged),
							Message: string(meshCommons.ResourceDeleted),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							ObservedGeneration: 1,
							Status:             metav1.ConditionTrue,
							Reason:             string(meshCommons.Successful),
							Message:            string(meshCommons.DependenciesResolved),
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
				},
			},
			expectedErr:  errors.New("mesh in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "update sdk mesh compartment id",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       meshName,
					Generation: 2,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(meshCommons.Successful),
								Message:            string(meshCommons.ResourceActive),
								ObservedGeneration: 1,
							},
						},
					},
					MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
						Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
					},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateActive,
						CompartmentId:  &compartmentId1,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				ChangeMeshCompartment: func(ctx context.Context, meshId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(meshCommons.LifecycleStateChanged),
							Message:            string(meshCommons.ResourceUpdating),
							ObservedGeneration: 2,
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
				},
			},
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
			expectedErr: nil,
		},
		{
			name: "update sdk mesh",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       meshName,
					Generation: 2,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(meshCommons.Successful),
								Message:            string(meshCommons.ResourceActive),
								ObservedGeneration: 1,
							},
						},
					},
					MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
						Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
					},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateActive,
						CompartmentId:  &compartmentId,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				UpdateMesh: func(ctx context.Context, mesh *sdk.Mesh) error {
					return nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(meshCommons.LifecycleStateChanged),
							Message:            string(meshCommons.ResourceUpdating),
							ObservedGeneration: 2,
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
				},
			},
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
			expectedErr: nil,
		},
		{
			name: "fail to create mesh in the control plane and store retry token",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				CreateMesh: func(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
			},
			expectOpcRetryToken: true,
			response:            &servicemanager.OSOKResponse{ShouldRequeue: true},
			expectedErr:         errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name: "Mesh created without error and clear retry token",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(meshCommons.Successful),
								Message:            "Dependencies resolved successfully",
								ObservedGeneration: 1,
							},
						},
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
					LastUpdatedTime: &metav1.Time{Time: timeNow},
					OpcRetryToken:   &opcRetryToken},
			},
			fields: fields{
				CreateMesh: func(ctx context.Context, mesh *sdk.Mesh, opcRetryToken *string) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateCreating,
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModePermissive,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectOpcRetryToken: false,
			response:            &servicemanager.OSOKResponse{ShouldRequeue: true},
		},
		{
			name: "update sdk mesh spec and compartment id",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       meshName,
					Generation: 2,
				},
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					TagResources: api.TagResources{
						FreeFormTags: map[string]string{},
						DefinedTags:  map[string]api.MapValue{},
					},
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: api.OCID(meshId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(meshCommons.Successful),
								Message:            string(meshCommons.ResourceActive),
								ObservedGeneration: 1,
							},
						},
					},
					MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
						Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
					},
					LastUpdatedTime: &metav1.Time{Time: timeNow},
				},
			},
			fields: fields{
				GetMesh: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateActive,
						CompartmentId:  &compartmentId1,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
						TimeCreated: &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated: &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				GetMeshNewCompartment: func(ctx context.Context, id *api.OCID) (*sdk.Mesh, error) {
					return &sdk.Mesh{
						DisplayName:    &meshName,
						Id:             &meshId,
						LifecycleState: sdk.MeshLifecycleStateActive,
						CompartmentId:  &compartmentId,
						FreeformTags:   map[string]string{},
						DefinedTags:    map[string]map[string]interface{}{},
						Mtls: &sdk.MeshMutualTransportLayerSecurity{
							Minimum: sdk.MutualTransportLayerSecurityModeStrict,
						},
					}, nil
				},
				UpdateMesh: func(ctx context.Context, mesh *sdk.Mesh) error {
					return nil
				},
				ChangeMeshCompartment: func(ctx context.Context, meshId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID(meshId),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(meshCommons.LifecycleStateChanged),
							Message:            string(meshCommons.ResourceUpdating),
							ObservedGeneration: 2,
						},
					},
				},
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeStrict,
				},
			},
			response:    &servicemanager.OSOKResponse{ShouldRequeue: true},
			expectedErr: nil,
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

			if tt.fields.GetMesh != nil {
				serviceMeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetMesh)
			}

			if tt.fields.CreateMesh != nil {
				serviceMeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateMesh)
			}

			if tt.fields.ChangeMeshCompartment != nil {
				serviceMeshClient.EXPECT().ChangeMeshCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeMeshCompartment)
			}

			if tt.fields.UpdateMesh != nil {
				serviceMeshClient.EXPECT().UpdateMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateMesh)
			}

			key := types.NamespacedName{Name: meshName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.mesh))
			response, err := m.CreateOrUpdate(ctx, tt.mesh, ctrl.Request{})

			if tt.fields.GetMeshNewCompartment != nil {
				serviceMeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetMeshNewCompartment)
				response, err = m.CreateOrUpdate(ctx, tt.mesh, ctrl.Request{})
			}

			curMesh := &servicemeshapi.Mesh{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curMesh))

			if tt.expectOpcRetryToken {
				assert.NotNil(t, curMesh.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curMesh.Status.OpcRetryToken)
			}

			if err == nil {
				if tt.response != nil {
					assert.True(t, cmp.Equal(tt.response.ShouldRequeue, response.ShouldRequeue))
				} else {
					opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
					assert.True(t, cmp.Equal(tt.expectedStatus, curMesh.Status, opts))
				}
			} else {
				assert.EqualError(t, err, tt.expectedErr.Error())
			}
		})
	}
}

func TestMeshFinalize(t *testing.T) {
	tests := []struct {
		name              string
		mesh              *servicemeshapi.Mesh
		vs                *servicemeshapi.VirtualService
		ap                *servicemeshapi.AccessPolicy
		ig                *servicemeshapi.IngressGateway
		expectedCondition []servicemeshapi.ServiceMeshCondition
		expectedErr       error
	}{
		{
			name: "successfully finalize mesh",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{meshCommons.MeshFinalizer},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
		},
		{
			name: "failed to finalize mesh because of associated ingress gateway",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{meshCommons.MeshFinalizer},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			ig: &servicemeshapi.IngressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ig-1",
					Namespace: "mesh-namespace",
				},
				Spec: servicemeshapi.IngressGatewaySpec{
					Mesh: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name:      "mesh",
							Namespace: "mesh-namespace",
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			expectedCondition: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             string(meshCommons.DependenciesNotResolved),
						Message:            "failed to update mesh",
						ObservedGeneration: 1,
					},
				},
			},
			expectedErr: errors.New("mesh has pending subresources to be deleted: ingressGateways"),
		},
		{
			name: "failed to finalize mesh because of associated access policy",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{meshCommons.MeshFinalizer},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			ap: &servicemeshapi.AccessPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ap-1",
					Namespace: "mesh-namespace",
				},
				Spec: servicemeshapi.AccessPolicySpec{
					Mesh: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name:      "mesh",
							Namespace: "mesh-namespace",
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			expectedCondition: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             string(meshCommons.DependenciesNotResolved),
						Message:            "failed to update mesh",
						ObservedGeneration: 1,
					},
				},
			},
			expectedErr: errors.New("mesh has pending subresources to be deleted: accessPolicies"),
		},
		{
			name: "failed to finalize mesh because of associated virtualService",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "mesh",
					Namespace:  "mesh-namespace",
					Finalizers: []string{meshCommons.MeshFinalizer},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vs-1",
					Namespace: "mesh-namespace",
				},
				Spec: servicemeshapi.VirtualServiceSpec{
					Mesh: servicemeshapi.RefOrId{
						ResourceRef: &servicemeshapi.ResourceRef{
							Name:      "mesh",
							Namespace: "mesh-namespace",
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId: "mesh-id",
				},
			},
			expectedCondition: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             string(meshCommons.DependenciesNotResolved),
						Message:            "failed to update mesh",
						ObservedGeneration: 1,
					},
				},
			},
			expectedErr: errors.New("mesh has pending subresources to be deleted: virtualServices"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := framework.NewFakeClientFramework(t)

			resourceManager := &ResourceManager{
				log:    loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
				client: f.K8sClient,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			assert.NoError(t, f.K8sClient.Create(ctx, tt.mesh))
			if tt.vs != nil {
				assert.NoError(t, f.K8sClient.Create(ctx, tt.vs))
			}
			if tt.ap != nil {
				assert.NoError(t, f.K8sClient.Create(ctx, tt.ap))
			}
			if tt.ig != nil {
				assert.NoError(t, f.K8sClient.Create(ctx, tt.ig))
			}
			_, err := m.Delete(ctx, tt.mesh)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
				assert.True(t, len(tt.mesh.ObjectMeta.Finalizers) == 0)
			} else {
				assert.EqualError(t, err, tt.expectedErr.Error())
				assert.True(t, len(tt.mesh.ObjectMeta.Finalizers) != 0)
			}
		})
	}
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		mesh *servicemeshapi.Mesh
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.Mesh
	}{
		{
			name: "mesh active condition updated with service mesh client error",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: api.OCID(compartmentId),
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.Mesh{
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.mesh).Build()

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			m.UpdateServiceMeshActiveStatus(ctx, tt.args.mesh, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.mesh, opts), "diff", cmp.Diff(tt.want, tt.args.mesh, opts))
		})
	}
}

func Test_UpdateServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		mesh *servicemeshapi.Mesh
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.Mesh
	}{
		{
			name: "mesh dependencies active condition updated with empty error message",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: api.OCID(compartmentId),
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.Mesh{
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.mesh).Build()

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.mesh, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.mesh, opts), "diff", cmp.Diff(tt.want, tt.args.mesh, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		mesh *servicemeshapi.Mesh
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.Mesh
	}{
		{
			name: "mesh configured condition updated with service mesh client error",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: api.OCID(compartmentId),
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.Mesh{
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
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
			name: "mesh configured condition updated with empty error message",
			args: args{
				mesh: &servicemeshapi.Mesh{
					Spec: servicemeshapi.MeshSpec{
						CompartmentId: api.OCID(compartmentId),
						CertificateAuthorities: []servicemeshapi.CertificateAuthority{
							{
								Id: api.OCID(certificateAuthorityId),
							},
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.Mesh{
				Spec: servicemeshapi.MeshSpec{
					CompartmentId: api.OCID(compartmentId),
					CertificateAuthorities: []servicemeshapi.CertificateAuthority{
						{
							Id: api.OCID(certificateAuthorityId),
						},
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.mesh).Build()

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Mesh")},
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.mesh, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.mesh, opts), "diff", cmp.Diff(tt.want, tt.args.mesh, opts))
		})
	}
}

func TestListVirtualServiceMembers(t *testing.T) {
	ms := &servicemeshapi.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mesh",
			Namespace: "mesh-namespace",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	vsInMesh_1 := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "mesh-namespace",
		},
		Spec: servicemeshapi.VirtualServiceSpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh",
					Namespace: "mesh-namespace",
				},
			},
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	vsNotInMesh_1 := &servicemeshapi.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "mesh-namespace",
			Name:      "vs-3",
		},
		Spec: servicemeshapi.VirtualServiceSpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh-1",
					Namespace: "mesh-namespace",
				},
			},
		},
	}

	type env struct {
		virtualServices []*servicemeshapi.VirtualService
	}
	type args struct {
		ms *servicemeshapi.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "found no virtualService",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{},
			},
			args: args{
				ms: ms,
			},
			want: false,
		},
		{
			name: "found virtualServices that matches",
			env: env{
				virtualServices: []*servicemeshapi.VirtualService{
					vsInMesh_1, vsNotInMesh_1,
				},
			},
			args: args{
				ms: ms,
			},
			want: true,
		},
	}
	f := framework.NewFakeClientFramework(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &ResourceManager{
				client: f.K8sClient,
			}

			for _, vs := range tt.env.virtualServices {
				err := f.K8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasVirtualServiceMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func TestListAccessPolicyMembers(t *testing.T) {
	ms := &servicemeshapi.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mesh",
			Namespace: "mesh-namespace",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	apInMesh_1 := &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ap-1",
			Namespace: "mesh-namespace",
		},
		Spec: servicemeshapi.AccessPolicySpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh",
					Namespace: "mesh-namespace",
				},
			},
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	apNotInMesh_1 := &servicemeshapi.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "mesh-namespace",
			Name:      "ap-3",
		},
		Spec: servicemeshapi.AccessPolicySpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh-1",
					Namespace: "mesh-namespace",
				},
			},
		},
	}

	type env struct {
		accessPolicies []*servicemeshapi.AccessPolicy
	}
	type args struct {
		ms *servicemeshapi.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "found no accessPolicy",
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{},
			},
			args: args{
				ms: ms,
			},
			want: false,
		},
		{
			name: "found accessPolicies that matches",
			env: env{
				accessPolicies: []*servicemeshapi.AccessPolicy{
					apInMesh_1, apNotInMesh_1,
				},
			},
			args: args{
				ms: ms,
			},
			want: true,
		},
	}
	f := framework.NewFakeClientFramework(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &ResourceManager{
				client: f.K8sClient,
			}

			for _, vs := range tt.env.accessPolicies {
				err := f.K8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasAccessPolicyMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func TestListIngressGatewayMembers(t *testing.T) {
	ms := &servicemeshapi.Mesh{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mesh",
			Namespace: "mesh-namespace",
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	igInMesh_1 := &servicemeshapi.IngressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ig-1",
			Namespace: "mesh-namespace",
		},
		Spec: servicemeshapi.IngressGatewaySpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh",
					Namespace: "mesh-namespace",
				},
			},
		},
		Status: servicemeshapi.ServiceMeshStatus{
			MeshId: "mesh-id",
		},
	}
	igNotInMesh_1 := &servicemeshapi.IngressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "mesh-namespace",
			Name:      "ig-3",
		},
		Spec: servicemeshapi.IngressGatewaySpec{
			Mesh: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name:      "mesh-1",
					Namespace: "mesh-namespace",
				},
			},
		},
	}

	type env struct {
		ingressGateways []*servicemeshapi.IngressGateway
	}
	type args struct {
		ms *servicemeshapi.Mesh
	}
	tests := []struct {
		name    string
		env     env
		args    args
		want    bool
		wantErr error
	}{
		{
			name: "found no ingressGateway",
			env: env{
				ingressGateways: []*servicemeshapi.IngressGateway{},
			},
			args: args{
				ms: ms,
			},
			want: false,
		},
		{
			name: "found ingressGateways that matches",
			env: env{
				ingressGateways: []*servicemeshapi.IngressGateway{
					igInMesh_1, igNotInMesh_1,
				},
			},
			args: args{
				ms: ms,
			},
			want: true,
		},
	}
	f := framework.NewFakeClientFramework(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &ResourceManager{
				client: f.K8sClient,
			}

			for _, vs := range tt.env.ingressGateways {
				err := f.K8sClient.Create(ctx, vs.DeepCopy())
				assert.NoError(t, err)
			}

			got, err := m.hasIngressGatewayMembers(ctx, tt.args.ms)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want, got, opts), "diff", cmp.Diff(tt.want, got, opts))
			}
		})
	}
}

func TestUpdateOpcRetryToken(t *testing.T) {
	tests := []struct {
		name                  string
		mesh                  *servicemeshapi.Mesh
		opcRetryToken         *string
		expectedOpcRetryToken *string
	}{
		{
			name: "add opc token for new request",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			opcRetryToken:         &opcRetryToken,
			expectedOpcRetryToken: &opcRetryToken,
		},
		{
			name: "delete opc token from status",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: meshName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:        api.OCID(meshId),
					OpcRetryToken: &opcRetryToken,
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

			key := types.NamespacedName{Name: meshName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.mesh))
			_ = m.UpdateOpcRetryToken(ctx, tt.mesh, tt.opcRetryToken)
			curMesh := &servicemeshapi.Mesh{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curMesh))
			assert.Same(t, tt.expectedOpcRetryToken, tt.opcRetryToken)
		})
	}
}
