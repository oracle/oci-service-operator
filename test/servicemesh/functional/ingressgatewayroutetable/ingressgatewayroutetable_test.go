/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewayroutetable

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	"github.com/oracle/oci-service-operator/test/servicemesh/functional"
)

var (
	testEnvFramework functional.TestEnvFramework
	testEnv          *envtest.Environment
	config           *rest.Config
	ctx              context.Context
)

func beforeEach(t *testing.T) *functional.Framework {
	ctx = context.Background()
	testEnvFramework = functional.NewDefaultTestEnvFramework()
	testEnv, config = testEnvFramework.SetupTestEnv()
	framework := testEnvFramework.SetupTestFramework(t, config)
	framework.CreateNamespace(ctx, "test-namespace")
	return framework
}

func afterEach(f *functional.Framework) {
	testEnvFramework.CleanUpTestFramework(f)
	testEnvFramework.CleanUpTestEnv(testEnv)
}

func TestIngressGatewayRouteTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		compartmentId  string
		freeformTags   map[string]string
		definedTagsSdk map[string]map[string]interface{}
		definedTags    map[string]api.MapValue
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
	}{
		{
			name: "create ingressGatewayRouteTable",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID("my-ingressgatewayroutetable-id"),
				IngressGatewayId:           api.OCID("my-ingressgateway-id"),
				IngressGatewayName:         servicemeshapi.Name("my-ingressgateway"),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-virtualservice-id"}},
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
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 1,
						},
					},
				},
			},
		},
		{
			name: "change ingressGatewayRouteTable compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID("my-ingressgatewayroutetable-id"),
				IngressGatewayId:           api.OCID("my-ingressgateway-id"),
				IngressGatewayName:         servicemeshapi.Name("my-ingressgateway"),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-virtualservice-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
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
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
		{
			name: "update ingressGatewayRouteTable freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID("my-ingressgatewayroutetable-id"),
				IngressGatewayId:           api.OCID("my-ingressgateway-id"),
				IngressGatewayName:         servicemeshapi.Name("my-ingressgateway"),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-virtualservice-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
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
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
		{
			name: "update ingressGatewayRouteTable definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				IngressGatewayRouteTableId: api.OCID("my-ingressgatewayroutetable-id"),
				IngressGatewayId:           api.OCID("my-ingressgateway-id"),
				IngressGatewayName:         servicemeshapi.Name("my-ingressgateway"),
				VirtualServiceIdForRules:   [][]api.OCID{{"my-virtualservice-id"}},
				Conditions: []servicemeshapi.ServiceMeshCondition{
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
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActive),
							ObservedGeneration: 2,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create the mesh
			mesh := functional.GetApiMesh()
			framework.MeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkMesh(sdk.MeshLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Create the ingressGateway
			ingressGateway := functional.GetApiIngressGateway()
			framework.MeshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGateway))

			virtualService := functional.GetApiVirtualService()
			framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

			ingressGatewayRouteTable := functional.GetApiIngressGatewayRouteTable()
			framework.MeshClient.EXPECT().CreateIngressGatewayRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayRouteTable))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkIngressGatewayRouteTable0 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable1 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateUpdating)
				sdkIngressGatewayRouteTable1.CompartmentId = &tt.args.compartmentId
				sdkIngressGatewayRouteTable2 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable2.CompartmentId = &tt.args.compartmentId
				sdkIngressGatewayRouteTable3 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateUpdating)
				sdkIngressGatewayRouteTable3.CompartmentId = &tt.args.compartmentId
				sdkIngressGatewayRouteTable4 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable2, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable3, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeIngressGatewayRouteTableCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGatewayRouteTable := ingressGatewayRouteTable.DeepCopy()
				ingressGatewayRouteTable.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, ingressGatewayRouteTable, oldIngressGatewayRouteTable)
				assert.NoError(t, err)
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkIngressGatewayRouteTable0 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable1 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateUpdating)
				sdkIngressGatewayRouteTable1.FreeformTags = tt.args.freeformTags
				sdkIngressGatewayRouteTable2 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGatewayRouteTable := ingressGatewayRouteTable.DeepCopy()
				ingressGatewayRouteTable.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, ingressGatewayRouteTable, oldIngressGatewayRouteTable)
				assert.NoError(t, err)
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkIngressGatewayRouteTable0 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable1 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateUpdating)
				sdkIngressGatewayRouteTable1.DefinedTags = tt.args.definedTagsSdk
				sdkIngressGatewayRouteTable2 := functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive)
				sdkIngressGatewayRouteTable2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(sdkIngressGatewayRouteTable2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGatewayRouteTable := ingressGatewayRouteTable.DeepCopy()
				ingressGatewayRouteTable.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, ingressGatewayRouteTable, oldIngressGatewayRouteTable)
				assert.NoError(t, err)
			}

			curIngressGatewayRouteTable := &servicemeshapi.IngressGatewayRouteTable{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(ingressGatewayRouteTable)
			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGatewayRouteTable))
			if tt.expectedStatus != nil {
				assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGatewayRouteTable.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGatewayRouteTable.Status, opts))
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGatewayRouteTable))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGateway))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
		})
		afterEach(framework)
	}
}
