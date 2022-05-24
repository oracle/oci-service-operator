/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualserviceroutetable

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	"github.com/oracle/oci-service-operator/test/servicemesh/functional"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
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

func TestVirtualServiceRouteTable(t *testing.T) {
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
			name: "create virtualServiceRouteTable",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				VirtualServiceRouteTableId:  api.OCID("my-virtualserviceroutetable-id"),
				VirtualServiceId:            api.OCID("my-virtualservice-id"),
				VirtualServiceName:          servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentIdForRules: [][]api.OCID{{"my-virtualdeployment-id"}},
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
			name: "change virtualServiceRouteTable compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				VirtualServiceRouteTableId:  api.OCID("my-virtualserviceroutetable-id"),
				VirtualServiceId:            api.OCID("my-virtualservice-id"),
				VirtualServiceName:          servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentIdForRules: [][]api.OCID{{"my-virtualdeployment-id"}},
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
			name: "update virtualServiceRouteTable freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				VirtualServiceRouteTableId:  api.OCID("my-virtualserviceroutetable-id"),
				VirtualServiceId:            api.OCID("my-virtualservice-id"),
				VirtualServiceName:          servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentIdForRules: [][]api.OCID{{"my-virtualdeployment-id"}},
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
			name: "update virtualServiceRouteTable definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				VirtualServiceRouteTableId:  api.OCID("my-virtualserviceroutetable-id"),
				VirtualServiceId:            api.OCID("my-virtualservice-id"),
				VirtualServiceName:          servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentIdForRules: [][]api.OCID{{"my-virtualdeployment-id"}},
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

			virtualService := functional.GetApiVirtualService()
			framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

			virtualDeployment := functional.GetApiVirtualDeployment()
			framework.MeshClient.EXPECT().CreateVirtualDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeployment))

			virtualServiceRouteTable := functional.GetApiVirtualServiceRouteTable()
			framework.MeshClient.EXPECT().CreateVirtualServiceRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualServiceRouteTable))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkVirtualServiceRouteTable0 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable1 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateUpdating)
				sdkVirtualServiceRouteTable1.CompartmentId = &tt.args.compartmentId
				sdkVirtualServiceRouteTable2 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable2.CompartmentId = &tt.args.compartmentId
				sdkVirtualServiceRouteTable3 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateUpdating)
				sdkVirtualServiceRouteTable3.CompartmentId = &tt.args.compartmentId
				sdkVirtualServiceRouteTable4 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable2, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable3, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeVirtualServiceRouteTableCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualServiceRouteTable := virtualServiceRouteTable.DeepCopy()
				virtualServiceRouteTable.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, virtualServiceRouteTable, oldVirtualServiceRouteTable)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualServiceRouteTable))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkVirtualServiceRouteTable0 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable1 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateUpdating)
				sdkVirtualServiceRouteTable1.FreeformTags = tt.args.freeformTags
				sdkVirtualServiceRouteTable2 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualServiceRouteTable := virtualServiceRouteTable.DeepCopy()
				virtualServiceRouteTable.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, virtualServiceRouteTable, oldVirtualServiceRouteTable)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualServiceRouteTable))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkVirtualServiceRouteTable0 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable1 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateUpdating)
				sdkVirtualServiceRouteTable1.DefinedTags = tt.args.definedTagsSdk
				sdkVirtualServiceRouteTable2 := functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive)
				sdkVirtualServiceRouteTable2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(sdkVirtualServiceRouteTable2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualServiceRouteTable := virtualServiceRouteTable.DeepCopy()
				virtualServiceRouteTable.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, virtualServiceRouteTable, oldVirtualServiceRouteTable)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualServiceRouteTable))
			}

			curVirtualServiceRouteTable := &servicemeshapi.VirtualServiceRouteTable{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(virtualServiceRouteTable)
			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualServiceRouteTable))
			if tt.expectedStatus != nil {
				assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualServiceRouteTable.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualServiceRouteTable.Status, opts))
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualServiceRouteTable))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeployment))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
		})
		afterEach(framework)
	}
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, virtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable) error {
	observedVirtualServiceRouteTable := &servicemeshapi.VirtualServiceRouteTable{}
	key := namespace.NewNamespacedName(virtualServiceRouteTable)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedVirtualServiceRouteTable); err != nil {
			return false, err
		}
		if observedVirtualServiceRouteTable != nil && commons.GetServiceMeshCondition(&observedVirtualServiceRouteTable.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
