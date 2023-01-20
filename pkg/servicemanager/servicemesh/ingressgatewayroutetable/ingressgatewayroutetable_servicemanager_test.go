/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewayroutetable

import (
	"context"
	"errors"
	"sort"
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
	igrtName           = "igrt"
	igrtId             = "igrt-id"
	compartmentId      = "compartment-id"
	ingressGatewayId   = "ingress-gateway"
	ingressGatewayName = "ingress-gateway-name"
	path               = "/foo"
	grpcEnabled        = true
	grpcDisabled       = false
	opcRetryToken      = "opcRetryToken"
	timeNow            = time.Now()
)

func TestOsokFinalize(t *testing.T) {
	type fields struct {
		DeleteIngressGatewayRouteTable func(ctx context.Context, igrtId *api.OCID) error
	}
	tests := []struct {
		name   string
		fields fields
		igrt   *servicemeshapi.IngressGatewayRouteTable
	}{
		{
			name: "sdk igrt deleted",
			fields: fields{
				DeleteIngressGatewayRouteTable: func(ctx context.Context, igrtId *api.OCID) error {
					return nil
				},
			},
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: api.OCID(igrtId),
				},
			},
		},
		{
			name: "sdk igrt not deleted",
			fields: fields{
				DeleteIngressGatewayRouteTable: nil,
			},
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{core.OSOKFinalizerName},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			defer controller.Finish()
			meshClient := meshMocks.NewMockServiceMeshClient(controller)
			testFramework := framework.NewFakeClientFramework(t)
			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IGRT")},
				client:            testFramework.K8sClient,
				serviceMeshClient: meshClient,
			}

			m := manager.NewServiceMeshServiceManager(testFramework.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.DeleteIngressGatewayRouteTable != nil {
				meshClient.EXPECT().DeleteIngressGatewayRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.DeleteIngressGatewayRouteTable)
			}

			_, err := m.Delete(ctx, tt.igrt)
			assert.True(t, len(tt.igrt.ObjectMeta.Finalizers) != 0)
			assert.NoError(t, err)
		})
	}
}

func TestGetIngressGatewayRouteTable(t *testing.T) {
	type fields struct {
		GetIngressGatewayRouteTable func(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGatewayRouteTable, error)
	}
	tests := []struct {
		name   string
		fields fields
		igrt   *servicemeshapi.IngressGatewayRouteTable
	}{
		{
			name: "sdk igrt created",
			fields: fields{
				GetIngressGatewayRouteTable: func(ctx context.Context, igrtId *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return nil, nil
				},
			},
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: api.OCID(igrtId),
				},
			},
		},
		{
			name: "sdk igrt not created",
			fields: fields{
				GetIngressGatewayRouteTable: nil,
			},
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: "",
				},
			},
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
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IGRT")},
			}

			if tt.fields.GetIngressGatewayRouteTable != nil {
				meshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIngressGatewayRouteTable)
			}

			err := m.GetResource(ctx, tt.igrt, &manager.ResourceDetails{})
			assert.NoError(t, err)
		})
	}
}

func TestUpdateServiceMeshCondition(t *testing.T) {
	tests := []struct {
		name               string
		igrt               *servicemeshapi.IngressGatewayRouteTable
		status             metav1.ConditionStatus
		expectedConditions []servicemeshapi.ServiceMeshCondition
		reason             string
		message            string
	}{
		{
			name: "api igrt has no condition and failed to get sdk igrt",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			status: metav1.ConditionFalse,
			expectedConditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionFalse,
						Reason:             "reason",
						Message:            "failed to get igrt",
						ObservedGeneration: 1,
					},
				},
			},
			reason:  "reason",
			message: "failed to get igrt",
		},
		{
			name: "api igrt has no condition and failed to update sdk igrt",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: api.OCID(igrtId),
				},
			},
			status: metav1.ConditionUnknown,
			expectedConditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             "reason",
						Message:            "failed to update igrt",
						ObservedGeneration: 1,
					},
				},
			},
			reason:  "reason",
			message: "failed to update igrt",
		},
		{
			name: "api igrt has condition and failed to delete igrt",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: api.OCID(igrtId),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionTrue,
								Reason:  "Successful",
								Message: "Successful",
							},
						},
					},
				},
			},
			status: metav1.ConditionUnknown,
			expectedConditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:  metav1.ConditionUnknown,
						Reason:  "reason",
						Message: "failed to cleanup igrt",
					},
				},
			},
			reason:  "reason",
			message: "failed to cleanup igrt",
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
				client:            f.K8sClient,
				serviceMeshClient: meshClient,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)
			key := types.NamespacedName{Name: igrtName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.igrt))
			_ = m.UpdateServiceMeshCondition(ctx, tt.igrt, tt.status, tt.reason, tt.message, servicemeshapi.ServiceMeshActive)
			curIngressGatewayRouteTable := &servicemeshapi.IngressGatewayRouteTable{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curIngressGatewayRouteTable))
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.expectedConditions, curIngressGatewayRouteTable.Status.Conditions, opts), "diff", cmp.Diff(tt.expectedConditions, curIngressGatewayRouteTable.Status.Conditions, opts))
		})
	}
}

func TestUpdateK8s(t *testing.T) {
	tests := []struct {
		name                        string
		igrt                        *servicemeshapi.IngressGatewayRouteTable
		sdkIngressGatewayRouteTable *sdk.IngressGatewayRouteTable
		expectedStatus              servicemeshapi.ServiceMeshStatus
		expectedErr                 error
	}{
		{
			name: "update igrt id and condition the first time",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			sdkIngressGatewayRouteTable: &sdk.IngressGatewayRouteTable{
				Name:             &igrtName,
				Id:               &igrtId,
				IngressGatewayId: &ingressGatewayId,
				LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateCreating,
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID(igrtId),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionUnknown,
							Reason:  string(commons.LifecycleStateChanged),
							Message: string(commons.ResourceCreating),
						},
					},
				},
			},
			expectedErr: errors.New(commons.UnknownStatus),
		},
		{
			name: "api igrt is created",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(commons.LifecycleStateChanged),
								Message: string(commons.ResourceCreating),
							},
						},
					},
				},
			},
			sdkIngressGatewayRouteTable: &sdk.IngressGatewayRouteTable{
				Name:             &igrtName,
				Id:               &igrtId,
				IngressGatewayId: &ingressGatewayId,
				LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateActive,
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID(igrtId),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
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
			expectedErr: nil,
		},
		{
			name: "sdk igrt is failed",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:  metav1.ConditionUnknown,
								Reason:  string(commons.LifecycleStateChanged),
								Message: string(commons.ResourceCreating),
							},
						},
					},
				},
			},
			sdkIngressGatewayRouteTable: &sdk.IngressGatewayRouteTable{
				Name:             &igrtName,
				Id:               &igrtId,
				IngressGatewayId: &ingressGatewayId,
				LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateFailed,
			},
			expectedStatus: servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID(igrtId),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
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
			expectedErr: nil,
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
				client:            f.K8sClient,
				serviceMeshClient: meshClient,
			}
			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)
			key := types.NamespacedName{Name: igrtName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.igrt))
			igrtDetails := &manager.ResourceDetails{}
			igrtDetails.IgrtDetails.Dependencies = &conversions.IGRTDependencies{
				IngressGatewayName: servicemeshapi.Name(ingressGatewayName),
				VsIdForRules:       make([][]api.OCID, 0),
			}
			igrtDetails.IgrtDetails.SdkIgrt = tt.sdkIngressGatewayRouteTable
			_, _ = m.UpdateK8s(ctx, tt.igrt, igrtDetails, false, false)
			curIngressGatewayRouteTable := &servicemeshapi.IngressGatewayRouteTable{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curIngressGatewayRouteTable))
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.expectedStatus, curIngressGatewayRouteTable.Status, opts), "diff", cmp.Diff(tt.expectedStatus, curIngressGatewayRouteTable.Status, opts))
		})
	}
}

func TestCreateOrUpdate(t *testing.T) {
	type fields struct {
		ResolveIngressGatewayIdAndNameAndMeshId   func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		ResolveVirtualServiceIdForRouteTable      []func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
		GetIngressGatewayRouteTable               func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error)
		GetIngressGatewayRouteTableNewCompartment func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error)
		CreateIngressGatewayRouteTable            func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error)
		UpdateIngressGatewayRouteTable            func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable) error
		DeleteIngressGatewayRouteTable            func(ctx context.Context, id *api.OCID) error
		ChangeIngressGatewayRouteTableCompartment func(ctx context.Context, igrtId *api.OCID, compartmentId *api.OCID) error
	}
	requestTimeout2000 := int64(2000)
	tests := []struct {
		name                                string
		igrt                                *servicemeshapi.IngressGatewayRouteTable
		fields                              fields
		returnedSdkIngressGatewayRouteTable *sdk.IngressGatewayRouteTable
		expectedStatus                      *servicemeshapi.ServiceMeshStatus
		expectedConditions                  []servicemeshapi.ServiceMeshCondition
		expectedErr                         error
		expectOpcRetryToken                 bool
		wantErr                             error
		doNotRequeue                        bool
	}{
		{
			name: "failed to resolve dependencies",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("failed to validate ingress gateway id")
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "failed to validate ingress gateway id",
							ObservedGeneration: 1,
						},
					},
				},
			},
			expectedErr: errors.New("failed to validate ingress gateway id"),
		},
		{
			name: "failed to get igrt",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 1,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.LifecycleStateChanged),
								Message:            string(commons.ResourceCreating),
								ObservedGeneration: 1,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return nil, meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89")
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.LifecycleStateChanged),
							Message:            string(commons.ResourceCreating),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            "Dependencies resolved successfully",
							ObservedGeneration: 1,
						},
					},
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
			expectedErr:  meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			doNotRequeue: true,
		},
		{
			name: "failed to create igrt",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				CreateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error) {
					return nil, meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89")
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            "Dependencies resolved successfully",
							ObservedGeneration: 1,
						},
					},
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
			expectedErr:  meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			doNotRequeue: true,
		},
		{
			name: "failed to create igrt and store retry token",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				CreateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error) {
					return nil, meshErrors.NewServiceError(500, "Internal error", "Internal error", "12-35-89")
				},
			},
			expectedConditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshDependenciesActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionTrue,
						Reason:             string(commons.Successful),
						Message:            "Dependencies resolved successfully",
						ObservedGeneration: 1,
					},
				},
				{
					Type: servicemeshapi.ServiceMeshConfigured,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             "Internal error",
						Message:            "Internal error (opc-request-id: 12-35-89 )",
						ObservedGeneration: 1,
					},
				},
			},
			expectOpcRetryToken: true,
			expectedErr:         errors.New("Service error:Internal error. Internal error. http status code: 500"),
		},
		{
			name: "Igrt created without error and clear retry token",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 1,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionTrue,
								Reason:             string(commons.Successful),
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
					OpcRetryToken: &opcRetryToken},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				CreateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateCreating,
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedConditions: []servicemeshapi.ServiceMeshCondition{
				{
					Type: servicemeshapi.ServiceMeshDependenciesActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionTrue,
						Reason:             string(commons.Successful),
						Message:            "Dependencies resolved successfully",
						ObservedGeneration: 1,
					},
				},
				{
					Type: servicemeshapi.ServiceMeshConfigured,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionTrue,
						Reason:             "Successful",
						Message:            "Resource configured successfully",
						ObservedGeneration: 1,
					},
				},
				{
					Type: servicemeshapi.ServiceMeshActive,
					ResourceCondition: servicemeshapi.ResourceCondition{
						Status:             metav1.ConditionUnknown,
						Reason:             "LifecycleStateChanged",
						Message:            "Resource in the control plane is Creating, about to reconcile",
						ObservedGeneration: 1,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "sdk igrt not created",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 1,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
					RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
									{
										VirtualService: &servicemeshapi.RefOrId{
											Id: "my-vd-id",
										},
									},
								},
								IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
									Name: "testHost",
								},
								Path:     &path,
								IsGrpc:   &grpcEnabled,
								PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name(ingressGatewayName),
					}
					return &ingressGatewayR, nil
				},
				ResolveVirtualServiceIdForRouteTable: []func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id:   api.OCID("my-vs-id"),
							Name: servicemeshapi.Name("my-vs-name"),
						}
						return &virtualServiceR, nil
					},
				},
				CreateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable, opcRetryToken *string) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateCreating,
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-vs-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            "Dependencies resolved successfully",
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.LifecycleStateChanged),
							Message:            string(commons.ResourceCreating),
							ObservedGeneration: 1,
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "sdk igrt is created and igrt spec was updated",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 2,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
					RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
									{
										VirtualService: &servicemeshapi.RefOrId{
											Id: "my-vd-id",
										},
									},
								},
								IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
									Name: "testHost",
								},
								Path:               &path,
								IsGrpc:             &grpcEnabled,
								PathType:           servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
								RequestTimeoutInMs: &requestTimeout2000,
							},
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								ObservedGeneration: 1,
								Status:             metav1.ConditionTrue,
								Reason:             string(commons.Successful),
								Message:            string(commons.ResourceActive),
							},
						},
					},
				},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name(ingressGatewayName),
					}
					return &ingressGatewayR, nil
				},
				ResolveVirtualServiceIdForRouteTable: []func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-vs1-id"),
						}
						return &virtualServiceR, nil
					},
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateActive,
						CompartmentId:    &compartmentId,
					}, nil
				},
				UpdateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable) error {
					return nil
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-vs1-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							ObservedGeneration: 2,
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.LifecycleStateChanged),
							Message:            string(commons.ResourceUpdating),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 2,
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "sdk igrt is deleted",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
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
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateDeleted,
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionFalse,
							Reason:  string(commons.LifecycleStateChanged),
							Message: string(commons.ResourceDeleted),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            "Dependencies resolved successfully",
							ObservedGeneration: 1,
						},
					},
				},
			},
			expectedErr:  errors.New("ingress gateway route table in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "Resolve dependencies error on create",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 2,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
					RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
									{
										VirtualService: &servicemeshapi.RefOrId{
											Id: "my-vd-id",
										},
									},
								},
								IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
									Name: "testHost",
								},
								Path:     &path,
								IsGrpc:   &grpcEnabled,
								PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					return nil, errors.New("error in resolving ingressgateway")
				},
			},
			expectedErr: errors.New("error in resolving ingressgateway"),
		},
		{
			name: "sdk igrt is failed",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name: igrtName,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
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
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateFailed,
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:  metav1.ConditionFalse,
							Reason:  string(commons.LifecycleStateChanged),
							Message: string(commons.ResourceFailed),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            "Dependencies resolved successfully",
							ObservedGeneration: 1,
						},
					},
				},
			},
			expectedErr:  errors.New("ingress gateway route table in the control plane is deleted or failed"),
			doNotRequeue: true,
		},
		{
			name: "update sdk igrt compartment id",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 2,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
					RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
									{
										VirtualService: &servicemeshapi.RefOrId{
											Id: "my-vd-id",
										},
									},
								},
								IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
									Name: "testHost",
								},
								Path:     &path,
								IsGrpc:   &grpcEnabled,
								PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								ObservedGeneration: 1,
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.LifecycleStateChanged),
								Message:            string(commons.ResourceCreating),
							},
						},
					},
				},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name("ig-name"),
					}
					return &ingressGatewayR, nil
				},
				ResolveVirtualServiceIdForRouteTable: []func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id:   api.OCID("my-vs-id"),
							Name: servicemeshapi.Name("my-vs-name"),
						}
						return &virtualServiceR, nil
					},
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateActive,
						CompartmentId:    conversions.String("compartment-2-id"),
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				ChangeIngressGatewayRouteTableCompartment: func(ctx context.Context, igrtId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			expectedErr: nil,
		},
		{
			name: "sdk igrt is created, igrt spec with compartmentId was updated",
			igrt: &servicemeshapi.IngressGatewayRouteTable{
				ObjectMeta: metav1.ObjectMeta{
					Name:       igrtName,
					Generation: 2,
				},
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					CompartmentId: api.OCID(compartmentId),
					IngressGateway: servicemeshapi.RefOrId{
						Id: api.OCID(ingressGatewayId),
					},
					RouteRules: []servicemeshapi.IngressGatewayTrafficRouteRule{
						{
							HttpRoute: &servicemeshapi.HttpIngressGatewayTrafficRouteRule{
								Destinations: []servicemeshapi.VirtualServiceTrafficRuleTarget{
									{
										VirtualService: &servicemeshapi.RefOrId{
											Id: "my-vd-id",
										},
									},
								},
								IngressGatewayHost: &servicemeshapi.IngressGatewayHostRef{
									Name: "testHost",
								},
								Path:     &path,
								IsGrpc:   &grpcEnabled,
								PathType: servicemeshapi.HttpIngressGatewayTrafficRouteRulePathTypePrefix,
							},
						},
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					IngressGatewayRouteTableId: (api.OCID(igrtId)),
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								ObservedGeneration: 1,
								Status:             metav1.ConditionTrue,
								Reason:             string(commons.Successful),
								Message:            string(commons.ResourceActive),
							},
						},
					},
				},
			},
			fields: fields{
				ResolveIngressGatewayIdAndNameAndMeshId: func(ctx context.Context, ResourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
					ingressGatewayR := commons.ResourceRef{
						Id:   api.OCID(ingressGatewayId),
						Name: servicemeshapi.Name(ingressGatewayName),
					}
					return &ingressGatewayR, nil
				},
				ResolveVirtualServiceIdForRouteTable: []func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error){
					func(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
						virtualServiceR := commons.ResourceRef{
							Id: api.OCID("my-vs1-id"),
						}
						return &virtualServiceR, nil
					},
				},
				GetIngressGatewayRouteTable: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateActive,
						CompartmentId:    conversions.String("compartment-2-id"),
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				GetIngressGatewayRouteTableNewCompartment: func(ctx context.Context, id *api.OCID) (*sdk.IngressGatewayRouteTable, error) {
					return &sdk.IngressGatewayRouteTable{
						Name:             &igrtName,
						Id:               &igrtId,
						IngressGatewayId: &ingressGatewayId,
						LifecycleState:   sdk.IngressGatewayRouteTableLifecycleStateActive,
						CompartmentId:    &compartmentId,
						TimeCreated:      &sdkcommons.SDKTime{Time: timeNow},
						TimeUpdated:      &sdkcommons.SDKTime{Time: timeNow},
					}, nil
				},
				UpdateIngressGatewayRouteTable: func(ctx context.Context, igrt *sdk.IngressGatewayRouteTable) error {
					return nil
				},
				ChangeIngressGatewayRouteTableCompartment: func(ctx context.Context, igrtId *api.OCID, compartmentId *api.OCID) error {
					return nil
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: (api.OCID(igrtId)),
				IngressGatewayId:           api.OCID(ingressGatewayId),
				IngressGatewayName:         servicemeshapi.Name(ingressGatewayName),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-vs1-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							ObservedGeneration: 2,
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.LifecycleStateChanged),
							Message:            string(commons.ResourceUpdating),
						},
					},
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.DependenciesResolved),
							ObservedGeneration: 2,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshConfigured,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceConfigured),
							ObservedGeneration: 2,
						},
					},
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
			f := framework.NewFakeClientFramework(t)
			meshClient := meshMocks.NewMockServiceMeshClient(controller)
			resolver := meshMocks.NewMockResolver(controller)

			resourceManager := &ResourceManager{
				log:               loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IG")},
				serviceMeshClient: meshClient,
				client:            f.K8sClient,
				referenceResolver: resolver,
			}

			m := manager.NewServiceMeshServiceManager(f.K8sClient, resourceManager.log, resourceManager)

			if tt.fields.ResolveIngressGatewayIdAndNameAndMeshId != nil {
				resolver.EXPECT().ResolveIngressGatewayIdAndNameAndMeshId(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ResolveIngressGatewayIdAndNameAndMeshId)
			}

			if tt.fields.ResolveVirtualServiceIdForRouteTable != nil {
				for _, resolveVirtualServiceIdForRouteTable := range tt.fields.ResolveVirtualServiceIdForRouteTable {
					resolver.EXPECT().ResolveVirtualServiceIdAndName(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(resolveVirtualServiceIdForRouteTable).AnyTimes()
				}
			}

			if tt.fields.GetIngressGatewayRouteTable != nil {
				meshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIngressGatewayRouteTable).Times(1)
			}

			if tt.fields.CreateIngressGatewayRouteTable != nil {
				meshClient.EXPECT().CreateIngressGatewayRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.CreateIngressGatewayRouteTable)
			}

			if tt.fields.UpdateIngressGatewayRouteTable != nil {
				meshClient.EXPECT().UpdateIngressGatewayRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.UpdateIngressGatewayRouteTable)
			}

			if tt.fields.ChangeIngressGatewayRouteTableCompartment != nil {
				meshClient.EXPECT().ChangeIngressGatewayRouteTableCompartment(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.ChangeIngressGatewayRouteTableCompartment)
			}

			key := types.NamespacedName{Name: igrtName}
			assert.NoError(t, f.K8sClient.Create(ctx, tt.igrt))
			var response servicemanager.OSOKResponse
			response, err := m.CreateOrUpdate(ctx, tt.igrt, ctrl.Request{})

			if tt.fields.GetIngressGatewayRouteTableNewCompartment != nil {
				meshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).DoAndReturn(tt.fields.GetIngressGatewayRouteTableNewCompartment).Times(1)
				response, err = m.CreateOrUpdate(ctx, tt.igrt, ctrl.Request{})
			}

			curIngressGatewayRouteTable := &servicemeshapi.IngressGatewayRouteTable{}
			assert.NoError(t, f.K8sClient.Get(ctx, key, curIngressGatewayRouteTable))

			if tt.expectOpcRetryToken {
				assert.NotNil(t, curIngressGatewayRouteTable.Status.OpcRetryToken)
			} else {
				assert.Nil(t, curIngressGatewayRouteTable.Status.OpcRetryToken)
			}

			if tt.expectedConditions != nil {
				trans := cmp.Transformer("Sort", func(in []servicemeshapi.ServiceMeshCondition) []servicemeshapi.ServiceMeshCondition {
					out := append([]servicemeshapi.ServiceMeshCondition(nil), in...) // Copy input to avoid mutating it
					sort.Slice(out, func(i, j int) bool {
						return out[i].Type < out[j].Type
					})
					return out
				})
				opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
				assert.True(t, cmp.Equal(tt.expectedConditions, curIngressGatewayRouteTable.Status.Conditions, opts, trans), "diff", cmp.Diff(tt.expectedConditions, curIngressGatewayRouteTable.Status.Conditions, opts, trans))
			}

			if tt.expectedStatus != nil {
				opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
				assert.True(t, cmp.Equal(tt.expectedStatus, &curIngressGatewayRouteTable.Status, opts), "diff", cmp.Diff(tt.expectedStatus, &curIngressGatewayRouteTable.Status, opts))
			}

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, !tt.doNotRequeue, response.ShouldRequeue)
		})
	}
}

func TestFinalize(t *testing.T) {
	m := &ResourceManager{}
	err := m.Finalize(context.Background(), nil)
	assert.NoError(t, err)
}

func Test_UpdateServiceMeshActiveStatus(t *testing.T) {
	type args struct {
		igrt *servicemeshapi.IngressGatewayRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGatewayRouteTable
	}{
		{
			name: "ingress gateway route table active condition updated with service mesh client error",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table active condition updated with service mesh client timeout",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.igrt).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IGRT")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshActiveStatus(ctx, tt.args.igrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.igrt, opts), "diff", cmp.Diff(tt.want, tt.args.igrt, opts))
		})
	}
}

func Test_UpdateServiceMeshDependenciesActiveStatus(t *testing.T) {
	type args struct {
		igrt *servicemeshapi.IngressGatewayRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGatewayRouteTable
	}{
		{
			name: "ingress gateway route table dependencies active condition updated with service mesh client error",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table dependencies active condition updated with service mesh client timeout",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table dependencies active condition updated with empty error message",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table dependencies active condition updated with k8s error message",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: errors.New("my-ig-id is not active yet"),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
					},
				},
				Status: servicemeshapi.ServiceMeshStatus{
					Conditions: []servicemeshapi.ServiceMeshCondition{
						{
							Type: servicemeshapi.ServiceMeshDependenciesActive,
							ResourceCondition: servicemeshapi.ResourceCondition{
								Status:             metav1.ConditionUnknown,
								Reason:             string(commons.DependenciesNotResolved),
								Message:            "my-ig-id is not active yet",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.igrt).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IGRT")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			m.UpdateServiceMeshDependenciesActiveStatus(ctx, tt.args.igrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.igrt, opts), "diff", cmp.Diff(tt.want, tt.args.igrt, opts))
		})
	}
}

func Test_UpdateServiceMeshConfiguredStatus(t *testing.T) {
	type args struct {
		igrt *servicemeshapi.IngressGatewayRouteTable
		err  error
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.IngressGatewayRouteTable
	}{
		{
			name: "ingress gateway route table configured condition updated with service mesh client error",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceError(400, "MissingParameter", "Missing Parameter in the body", "12-35-89"),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table configured condition updated with service mesh client timeout",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
				err: meshErrors.NewServiceTimeoutError(),
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			name: "ingress gateway route table configured condition updated with empty error message",
			args: args{
				igrt: &servicemeshapi.IngressGatewayRouteTable{
					Spec: servicemeshapi.IngressGatewayRouteTableSpec{
						IngressGateway: servicemeshapi.RefOrId{
							Id: "my-ig-id",
						},
					},
					Status: servicemeshapi.ServiceMeshStatus{},
				},
			},
			want: &servicemeshapi.IngressGatewayRouteTable{
				Spec: servicemeshapi.IngressGatewayRouteTableSpec{
					IngressGateway: servicemeshapi.RefOrId{
						Id: "my-ig-id",
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
			k8sClient := testclient.NewClientBuilder().WithScheme(k8sSchema).WithObjects(tt.args.igrt).Build()
			resourceManager := &ResourceManager{
				log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("IGRT")},
			}
			m := manager.NewServiceMeshServiceManager(k8sClient, resourceManager.log, resourceManager)
			_ = m.UpdateServiceMeshConfiguredStatus(ctx, tt.args.igrt, tt.args.err)
			opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
			assert.True(t, cmp.Equal(tt.want, tt.args.igrt, opts), "diff", cmp.Diff(tt.want, tt.args.igrt, opts))
		})
	}
}
