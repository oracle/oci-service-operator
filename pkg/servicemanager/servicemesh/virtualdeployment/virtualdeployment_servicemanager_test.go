/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

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
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	mocks "github.com/oracle/oci-service-operator/test/mocks/servicemesh"
	meshErrors "github.com/oracle/oci-service-operator/test/servicemesh/errors"
)

var (
	vdDescription = servicemeshapi.Description("This is Virtual Deployment")
	opcRetryToken = "opcRetryToken"
	timeNow       = time.Now()
)

func Test_CreateUpdateChangeCompartmentForVD(t *testing.T) {
	outputVd := &sdk.VirtualDeployment{
		Id:               conversions.String("myVd"),
		VirtualServiceId: conversions.String("vsId"),
		Description:      conversions.String("This is Virtual Deployment"),
		LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
	}

	bFalse := false
	bTrue := true
	type functions struct {
		ResolveVSId       func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		create            func(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error)
		update            func(ctx context.Context, vd *sdk.VirtualDeployment) error
		changeCompartment func(ctx context.Context, vd *api.OCID, compartmentId *api.OCID) error
		get               func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error)
		getNewCompartment func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error)
	}
	tests := []struct {
		name                string
		functions           functions
		vd                  *servicemeshapi.VirtualDeployment
		vs                  *servicemeshapi.VirtualService
		times               int
		wantErr             error
		expectOpcRetryToken bool
	}{
		{
			name: "happy create path",
			vd:   getDefaultVd("", "vdCompartment", servicemeshapi.AccessLogging{}),
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-vs",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				create: func(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error) {
					return outputVd, nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "fail to create vd in the control plane",
			vd:   getDefaultVd("", "vdCompartment", servicemeshapi.AccessLogging{}),
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-vs",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				create: func(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
			},
			times:               1,
			expectOpcRetryToken: true,
			wantErr:             errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name: "vd created without error and clear retry token",
			vd:   getDefaultVd("", "vdCompartment", servicemeshapi.AccessLogging{}),
			vs: &servicemeshapi.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-vs",
				},
				Status: servicemeshapi.ServiceMeshStatus{
					OpcRetryToken: &opcRetryToken},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				create: func(ctx context.Context, vd *sdk.VirtualDeployment, opcRetryToken *string) (*sdk.VirtualDeployment, error) {
					return outputVd, nil
				},
			},
			times:               1,
			expectOpcRetryToken: false,
		},
		{
			name: "Resolve dependencies error on create",
			vd:   getDefaultVd("", "vdCompartment", servicemeshapi.AccessLogging{}),
			vs: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("error in resolving virtualservice")
				},
			},
			times:   1,
			wantErr: errors.New("error in resolving virtualservice"),
		},
		{
			name: "happy update path",
			vd: getDefaultVdWithStatus("myvd", "vdCompartment", servicemeshapi.AccessLogging{
				IsEnabled: true,
			}),
			vs: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				update: func(ctx context.Context, vd *sdk.VirtualDeployment) error {
					assert.Equal(t, vd.AccessLogging, &sdk.AccessLoggingConfiguration{
						IsEnabled: &bTrue,
					})
					return nil
				},
				get: func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						CompartmentId: conversions.String("vdCompartment"),
						AccessLogging: &sdk.AccessLoggingConfiguration{
							IsEnabled: &bFalse,
						},
						LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
						VirtualServiceId: conversions.String("myVs"),
						Id:               conversions.String("myVd"),
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			times:   2,
			wantErr: nil,
		},
		{
			name: "Happy change compartment path",
			vd:   getVdWithUpdatedSpec("myvd", "newCompartment", servicemeshapi.AccessLogging{}),
			vs: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				changeCompartment: func(ctx context.Context, vd *api.OCID, compartmentId *api.OCID) error {
					assert.Equal(t, *compartmentId, api.OCID("newCompartment"))
					return nil
				},
				get: func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						CompartmentId: conversions.String("vdCompartment"),
						AccessLogging: &sdk.AccessLoggingConfiguration{
							IsEnabled: &bFalse,
						},
						LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
						VirtualServiceId: conversions.String("myVs"),
						Id:               conversions.String("myVd"),
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "update deleted vd",
			vd: getDefaultVd("myvd", "vdCompartment", servicemeshapi.AccessLogging{
				IsEnabled: true,
			}),
			vs: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "vsId",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("vsId"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				get: func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						LifecycleState: sdk.VirtualDeploymentLifecycleStateDeleted,
						TimeCreated:    &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:    &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			times:   1,
			wantErr: nil,
		},
		{
			name: "change compartment with access logging",
			vd: getVdWithUpdatedSpec("myVd", "newCompartment", servicemeshapi.AccessLogging{
				IsEnabled: false,
			}),
			vs: &servicemeshapi.VirtualService{
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualServiceId: "my-vs-id",
				},
			},
			functions: functions{
				ResolveVSId: func(ctx context.Context, vsRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					virtualServiceRef := commons.ResourceRef{
						Id:   api.OCID("my-vs-id"),
						Name: servicemeshapi.Name("vsName"),
					}
					return &virtualServiceRef, nil
				},
				changeCompartment: func(ctx context.Context, vd *api.OCID, compartmentId *api.OCID) error {
					assert.Equal(t, *compartmentId, api.OCID("newCompartment"))
					return nil
				},
				update: func(ctx context.Context, vd *sdk.VirtualDeployment) error {
					assert.Equal(t, vd.AccessLogging, &sdk.AccessLoggingConfiguration{
						IsEnabled: &bFalse,
					})
					return nil
				},
				get: func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						CompartmentId: conversions.String("vdCompartment"),
						AccessLogging: &sdk.AccessLoggingConfiguration{
							IsEnabled: &bTrue,
						},
						LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
						VirtualServiceId: conversions.String("my-vs-id"),
						Id:               conversions.String("myVd"),
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				getNewCompartment: func(ctx context.Context, virtualDeploymentId *api.OCID) (*sdk.VirtualDeployment, error) {
					return &sdk.VirtualDeployment{
						CompartmentId: conversions.String("newCompartment"),
						AccessLogging: &sdk.AccessLoggingConfiguration{
							IsEnabled: &bTrue,
						},
						LifecycleState:   sdk.VirtualDeploymentLifecycleStateActive,
						VirtualServiceId: conversions.String("my-vs-id"),
						Id:               conversions.String("myVd"),
					}, nil
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
			meshClient := mocks.NewMockServiceMeshClient(controller)
			resolver := mocks.NewMockResolver(controller)

			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.vs, tt.vd).Build()

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")},
				serviceMeshClient: meshClient,
				client:            k8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)

			if tt.functions.ResolveVSId != nil {
				resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.ResolveVSId).AnyTimes()
			}
			if tt.functions.create != nil {
				meshClient.EXPECT().CreateVirtualDeployment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.create)
			}
			if tt.functions.update != nil {
				meshClient.EXPECT().UpdateVirtualDeployment(gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.update)
			}
			if tt.functions.changeCompartment != nil {
				meshClient.EXPECT().ChangeVirtualDeploymentCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.changeCompartment)
			}
			if tt.functions.get != nil {
				meshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.get).Times(tt.times)
			}

			for i := 0; i < tt.times; i++ {
				_, err = m.CreateOrUpdate(ctx, tt.vd, ctrl.Request{})
			}

			if tt.functions.getNewCompartment != nil {
				meshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).DoAndReturn(tt.functions.getNewCompartment).Times(1)
				_, err = m.CreateOrUpdate(ctx, tt.vd, ctrl.Request{})
			}

			key := types.NamespacedName{Name: "test"}
			curVd := &servicemeshapi.VirtualDeployment{}
			assert.NoError(t, k8sClient.Get(ctx, key, curVd))

			if tt.expectOpcRetryToken {
				assert.NotNil(t, curVd.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curVd.Status.OpcRetryToken)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Finalize(t *testing.T) {

	tests := []struct {
		name    string
		vd      *servicemeshapi.VirtualDeployment
		vdb     *servicemeshapi.VirtualDeploymentBinding
		vsrt    *servicemeshapi.VirtualServiceRouteTable
		wantErr error
	}{
		{
			name: "Has vdb dependencies",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{
					VirtualDeployment: servicemeshapi.RefOrId{
						//VirtualDeploymentRef: api.ResourceRef{
						//	Name:      "myVd",
						//	Namespace: "myNs",
						//	Id:        "123",
						//},
						Id: api.OCID("123"),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:              "",
					VirtualDeploymentId: "123",
					Conditions:          nil,
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myVd",
					Namespace: "myNs",
				},
				Spec: servicemeshapi.VirtualDeploymentSpec{},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "123",
				},
			},
			wantErr: errors.New("cannot delete virtual deployment when there are virtual deployment binding resources associated"),
		},
		{
			name: "Has vsrt dependencies",
			vdb: &servicemeshapi.VirtualDeploymentBinding{
				Spec: servicemeshapi.VirtualDeploymentBindingSpec{
					VirtualDeployment: servicemeshapi.RefOrId{
						Id: api.OCID("123"),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					MeshId:     "",
					Conditions: nil,
				},
			},
			vsrt: &servicemeshapi.VirtualServiceRouteTable{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myVsrt",
					Namespace: "myNs",
				},
				Spec: servicemeshapi.VirtualServiceRouteTableSpec{},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentIdForRules: getVdIdForRules("123"),
				},
			},
			vd: &servicemeshapi.VirtualDeployment{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myVd",
					Namespace: "myNs",
				},
				Spec: servicemeshapi.VirtualDeploymentSpec{},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "123",
				},
			},
			wantErr: errors.New("cannot delete virtual deployment when there are virtual service route table resources associated"),
		},
		{
			name: "Finalize the happy path",
			vdb:  &servicemeshapi.VirtualDeploymentBinding{},
			vd: &servicemeshapi.VirtualDeployment{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myVd",
					Namespace: "myNs",
				},
				Spec: servicemeshapi.VirtualDeploymentSpec{},
				Status: servicemeshapi.ServiceMeshStatus{
					VirtualDeploymentId: "test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup basics
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = servicemeshapi.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.vdb).Build()

			m := &ResourceManager{
				client: k8sClient,
			}
			if tt.vsrt != nil {
				err = k8sClient.Create(ctx, tt.vsrt.DeepCopy())
				assert.NoError(t, err)
			}

			if err := m.Finalize(ctx, tt.vd); err != nil {
				if tt.wantErr == nil {
					assert.Fail(t, "unexpected error")
				} else {
					assert.Equal(t, tt.wantErr, err)
				}
			} else {
				if tt.wantErr != nil {
					assert.Fail(t, "expected an error")
				}
			}
		})
	}
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		vd  *servicemeshapi.VirtualDeployment
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualDeployment
	}{
		{
			name: "virtual deployment active condition updated with service mesh client error",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment active condition updated with service mesh client timeout",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vd).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.vd, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vd, opts), "diff", cmp.Diff(tt.want, tt.args.vd, opts))
		})
	}
}

func Test_UpdateServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		vd  *servicemeshapi.VirtualDeployment
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualDeployment
	}{
		{
			name: "virtual deployment dependencies active condition updated with service mesh client error",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment dependencies active condition updated with service mesh client timeout",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment dependencies active condition updated with empty error message",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment dependencies active condition updated with k8s error message",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-vs-id is not active yet"),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.DependenciesNotResolved),
								Message:            "my-vs-id is not active yet",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vd).Build()

			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.vd, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vd, opts), "diff", cmp.Diff(tt.want, tt.args.vd, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		vd  *servicemeshapi.VirtualDeployment
		err error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualDeployment
	}{
		{
			name: "virtual deployment configured condition updated with service mesh client error",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment configured condition updated with service mesh client timeout",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			name: "virtual deployment configured condition updated with empty error message",
			args: args{
				vd: &servicemeshapi.VirtualDeployment{
					Spec: servicemeshapi.VirtualDeploymentSpec{
						VirtualService: servicemeshapi.RefOrId{
							Id: "my-vs-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.VirtualDeployment{
				Spec: servicemeshapi.VirtualDeploymentSpec{
					VirtualService: servicemeshapi.RefOrId{
						Id: "my-vs-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.vd).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.vd, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.vd, opts), "diff", cmp.Diff(tt.want, tt.args.vd, opts))
		})
	}
}

func getDefaultVd(id string, compartment string, logging servicemeshapi.AccessLogging) *servicemeshapi.VirtualDeployment {
	vd := servicemeshapi.VirtualDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: servicemeshapi.VirtualDeploymentSpec{
			CompartmentId: "my-compartment",
			Description:   &vdDescription,
			VirtualService: servicemeshapi.RefOrId{
				ResourceRef: &servicemeshapi.ResourceRef{
					Name: "my-vs",
				},
			},
		},
	}
	if id != "" {
		vd.Status.VirtualDeploymentId = api.OCID(id)
	}
	if compartment != "" {
		vd.Spec.CompartmentId = api.OCID(compartment)
	}
	vd.Spec.AccessLogging = &logging
	return &vd
}

func getDefaultVdWithStatus(id string, compartment string, logging servicemeshapi.AccessLogging) *servicemeshapi.VirtualDeployment {
	vd := getDefaultVd(id, compartment, logging)
	vd.Status = servicemeshapi.ServiceMeshStatus{
		VirtualDeploymentId: api.OCID(id),
		VirtualServiceId:    "my-vs-id",
	}
	return vd
}

func getVdWithUpdatedSpec(id string, compartment string, logging servicemeshapi.AccessLogging) *servicemeshapi.VirtualDeployment {
	vd := getDefaultVdWithStatus(id, compartment, logging)
	vd.Generation = 2
	newCondition := servicemeshapi.ServiceMeshCondition{
		Type: servicemeshapi.ServiceMeshActive,
		ResourceCondition: servicemeshapi.ResourceCondition{
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	vd.Status.Conditions = append(vd.Status.Conditions, newCondition)
	return vd
}

func getVdIdForRules(id string) [][]api.OCID {
	vdIds := make([][]api.OCID, 1)
	vdIds[0] = make([]api.OCID, 1)
	vdIds[0][0] = api.OCID(id)
	return vdIds
}
