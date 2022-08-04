/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

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
	"github.com/oracle/oci-service-operator/test/servicemesh/errors"
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

func TestVirtualDeployment(t *testing.T) {
	type args struct {
		compartmentId            string
		freeformTags             map[string]string
		definedTagsSdk           map[string]map[string]interface{}
		definedTags              map[string]api.MapValue
		virtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable
		virtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
		cpError                  error
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create virtualDeployment",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
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
			name: "change virtualDeployment compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
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
			name: "update virtualDeployment freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
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
			name: "update virtualDeployment definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
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
			name: "delete virtualDeployment failed because of associated virtualServiceRouteTables",
			args: args{
				virtualServiceRouteTable: functional.GetApiVirtualServiceRouteTable(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual deployment when there are virtual service route table resources associated",
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
			name: "delete virtualDeployment failed because of associated virtualDeploymentBinding",
			args: args{
				virtualDeploymentBinding: functional.GetApiVirtualDeploymentBinding(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual deployment when there are virtual deployment binding resources associated",
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
			name: "delete virtualDeployment failed because of associated dependencies in control plane",
			args: args{
				cpError: errors.NewServiceError(409, "Conflict", "IncorrectState. Can't delete the resource with dependencies", "123"),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:              api.OCID("my-mesh-id"),
				VirtualServiceId:    api.OCID("my-virtualservice-id"),
				VirtualServiceName:  servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentId: api.OCID("my-virtualdeployment-id"),
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
							ObservedGeneration: 1,
						},
					},
					{
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             "Conflict",
							Message:            "IncorrectState. Can't delete the resource with dependencies (opc-request-id: 123 )",
							ObservedGeneration: 1,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		if testing.Short() {
			t.Skip("Skipping Service Mesh Functional Tests during build.")
		}
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create the mesh
			mesh := functional.GetApiMesh()
			framework.MeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkMesh(sdk.MeshLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Create the virtualService
			virtualService := functional.GetApiVirtualService()
			framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
			framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

			// Create the virtualDeployment
			virtualDeployment := functional.GetApiVirtualDeployment()
			framework.MeshClient.EXPECT().CreateVirtualDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive), nil).AnyTimes()
			if tt.args.cpError != nil {
				framework.MeshClient.EXPECT().DeleteVirtualDeployment(gomock.Any(), gomock.Any()).Return(tt.args.cpError).AnyTimes()
			} else {
				framework.MeshClient.EXPECT().DeleteVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeployment))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkVirtualDeployment0 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment1 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateUpdating)
				sdkVirtualDeployment1.CompartmentId = &tt.args.compartmentId
				sdkVirtualDeployment2 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment2.CompartmentId = &tt.args.compartmentId
				sdkVirtualDeployment3 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateUpdating)
				sdkVirtualDeployment3.CompartmentId = &tt.args.compartmentId
				sdkVirtualDeployment4 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment2, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment3, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeVirtualDeploymentCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualDeployment := virtualDeployment.DeepCopy()
				virtualDeployment.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, virtualDeployment, oldVirtualDeployment)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualDeployment))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkVirtualDeployment0 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment1 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateUpdating)
				sdkVirtualDeployment1.FreeformTags = tt.args.freeformTags
				sdkVirtualDeployment2 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualDeployment := virtualDeployment.DeepCopy()
				virtualDeployment.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, virtualDeployment, oldVirtualDeployment)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualDeployment))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkVirtualDeployment0 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment1 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateUpdating)
				sdkVirtualDeployment1.DefinedTags = tt.args.definedTagsSdk
				sdkVirtualDeployment2 := functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive)
				sdkVirtualDeployment2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualDeployment(gomock.Any(), gomock.Any()).Return(sdkVirtualDeployment2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualDeployment := virtualDeployment.DeepCopy()
				virtualDeployment.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, virtualDeployment, oldVirtualDeployment)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualDeployment))
			}

			curVirtualDeployment := &servicemeshapi.VirtualDeployment{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(virtualDeployment)

			if tt.args.virtualServiceRouteTable != nil {
				// For virtualDeployment deletion fails with associated virtualServiceRouteTable test case, create virtualServiceRouteTable and validate the deletion
				virtualServiceRouteTable := tt.args.virtualServiceRouteTable.DeepCopy()
				framework.MeshClient.EXPECT().CreateVirtualServiceRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualServiceRouteTable))

				go func() {
					framework.K8sAPIs.Delete(ctx, virtualDeployment)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualDeployment))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualDeployment))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualDeployment.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualServiceRouteTable))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualDeployment)
			} else if tt.args.virtualDeploymentBinding != nil {
				// For virtualDeployment deletion fails with associated virtualDeploymentBinding test case, create virtualDeploymentBinding and validate the deletion
				namespace := functional.GetSidecarInjectNamespace()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, namespace))

				service := functional.GetService()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, service))

				virtualDeploymentBinding := tt.args.virtualDeploymentBinding.DeepCopy()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeploymentBinding))

				go func() {
					framework.K8sAPIs.Delete(ctx, virtualDeployment)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualDeployment))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualDeployment))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualDeployment.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeploymentBinding))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualDeployment)
			} else if tt.args.cpError != nil {
				go func() {
					framework.K8sAPIs.Delete(ctx, virtualDeployment)
				}()
				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualDeployment))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualDeployment))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualDeployment.Status, opts))
				}
			} else {
				// For all other test cases
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualDeployment))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualDeployment.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualDeployment.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeployment))
			}
		})
		afterEach(framework)
	}
}

func waitUntilStatusChanged(ctx context.Context, framework *functional.Framework, virtualDeployment *servicemeshapi.VirtualDeployment) error {
	observedVirtualDeployment := &servicemeshapi.VirtualDeployment{}
	key := namespace.NewNamespacedName(virtualDeployment)
	if err := framework.K8sClient.Get(ctx, key, observedVirtualDeployment); err != nil {
		return err
	}
	oldStatus := commons.GetServiceMeshCondition(&observedVirtualDeployment.Status, servicemeshapi.ServiceMeshActive).Status
	oldDependenciesStatus := commons.GetServiceMeshCondition(&observedVirtualDeployment.Status, servicemeshapi.ServiceMeshDependenciesActive).Status
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedVirtualDeployment); err != nil {
			return false, err
		}
		if observedVirtualDeployment != nil && commons.GetServiceMeshCondition(&observedVirtualDeployment.Status, servicemeshapi.ServiceMeshActive).Status != oldStatus {
			return true, nil
		}
		if observedVirtualDeployment != nil && commons.GetServiceMeshCondition(&observedVirtualDeployment.Status, servicemeshapi.ServiceMeshDependenciesActive).Status != oldDependenciesStatus {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, virtualDeployment *servicemeshapi.VirtualDeployment) error {
	observedVirtualDeployment := &servicemeshapi.VirtualDeployment{}
	key := namespace.NewNamespacedName(virtualDeployment)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedVirtualDeployment); err != nil {
			return false, err
		}
		if observedVirtualDeployment != nil && commons.GetServiceMeshCondition(&observedVirtualDeployment.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
