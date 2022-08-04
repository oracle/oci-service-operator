/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mesh

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

func TestMesh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		compartmentId  string
		freeformTags   map[string]string
		definedTagsSdk map[string]map[string]interface{}
		definedTags    map[string]api.MapValue
		virtualService *servicemeshapi.VirtualService
		ingressGateway *servicemeshapi.IngressGateway
		accessPolicy   *servicemeshapi.AccessPolicy
		cpError        error
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create mesh",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
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
			name: "change mesh compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
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
			name: "update mesh freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
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
			name: "update mesh definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
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
			name: "delete mesh failed because of associated virtualService",
			args: args{
				virtualService: functional.GetApiVirtualService(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "mesh has pending subresources to be deleted: virtualServices",
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
			name: "delete mesh failed because of associated ingressGateway",
			args: args{
				ingressGateway: functional.GetApiIngressGateway(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "mesh has pending subresources to be deleted: ingressGateways",
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
			name: "delete mesh failed because of associated accessPolicy",
			args: args{
				accessPolicy: functional.GetApiAccessPolicy(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "mesh has pending subresources to be deleted: virtualServices, accessPolicies",
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
			name: "delete mesh failed because of associated dependencies in control plane",
			args: args{
				cpError: errors.NewServiceError(409, "Conflict", "IncorrectState. Can't delete the resource with dependencies", "123"),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId: api.OCID("my-mesh-id"),
				MeshMtls: &servicemeshapi.MeshMutualTransportLayerSecurity{
					Minimum: servicemeshapi.MutualTransportLayerSecurityModeDisabled,
				},
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
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create the mesh
			mesh := functional.GetApiMesh()
			framework.MeshClient.EXPECT().CreateMesh(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkMesh(sdk.MeshLifecycleStateActive), nil).AnyTimes()
			if tt.args.cpError != nil {
				framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(tt.args.cpError).AnyTimes()
			} else {
				framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkMesh0 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh1 := functional.GetSdkMesh(sdk.MeshLifecycleStateUpdating)
				sdkMesh1.CompartmentId = &tt.args.compartmentId
				sdkMesh2 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh2.CompartmentId = &tt.args.compartmentId
				sdkMesh3 := functional.GetSdkMesh(sdk.MeshLifecycleStateUpdating)
				sdkMesh3.CompartmentId = &tt.args.compartmentId
				sdkMesh4 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh0, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh1, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh2, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh3, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeMeshCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateMesh(gomock.Any(), gomock.Any()).Return(nil)
				oldMesh := mesh.DeepCopy()
				mesh.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, mesh, oldMesh)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, mesh))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkMesh0 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh1 := functional.GetSdkMesh(sdk.MeshLifecycleStateUpdating)
				sdkMesh1.FreeformTags = tt.args.freeformTags
				sdkMesh2 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh0, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh1, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateMesh(gomock.Any(), gomock.Any()).Return(nil)
				oldMesh := mesh.DeepCopy()
				mesh.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, mesh, oldMesh)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, mesh))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkMesh0 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh1 := functional.GetSdkMesh(sdk.MeshLifecycleStateUpdating)
				sdkMesh1.DefinedTags = tt.args.definedTagsSdk
				sdkMesh2 := functional.GetSdkMesh(sdk.MeshLifecycleStateActive)
				sdkMesh2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh0, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh1, nil).Times(1),
					framework.MeshClient.EXPECT().GetMesh(gomock.Any(), gomock.Any()).Return(sdkMesh2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateMesh(gomock.Any(), gomock.Any()).Return(nil)
				oldMesh := mesh.DeepCopy()
				mesh.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, mesh, oldMesh)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, mesh))
			}

			curMesh := &servicemeshapi.Mesh{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(mesh)

			if tt.args.virtualService != nil {
				// For mesh deletion fails with associated virtualService test case, create virtualService and validate the deletion
				virtualService := tt.args.virtualService.DeepCopy()
				framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))
				go func() {
					framework.K8sAPIs.Delete(ctx, mesh)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, mesh))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curMesh))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curMesh.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curMesh.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
				framework.K8sAPIs.WaitUntilDeleted(ctx, mesh)
			} else if tt.args.ingressGateway != nil {
				ingressGateway := tt.args.ingressGateway.DeepCopy()
				// For mesh deletion fails with associated ingressGateway test case, create ingressGateway and validate the deletion
				framework.MeshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGateway))

				go func() {
					framework.K8sAPIs.Delete(ctx, mesh)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, mesh))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curMesh))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curMesh.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curMesh.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGateway))
				framework.K8sAPIs.WaitUntilDeleted(ctx, mesh)
			} else if tt.args.accessPolicy != nil {
				// For mesh deletion fails with associated accessPolicy test case, create accessPolicy and validate the deletion
				virtualService := functional.GetApiVirtualService()
				framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

				accessPolicy := tt.args.accessPolicy.DeepCopy()
				framework.MeshClient.EXPECT().CreateAccessPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteAccessPolicy(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, accessPolicy))

				go func() {
					framework.K8sAPIs.Delete(ctx, mesh)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, mesh))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curMesh))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curMesh.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curMesh.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, accessPolicy))
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
				framework.K8sAPIs.WaitUntilDeleted(ctx, mesh)
			} else if tt.args.cpError != nil {
				go func() {
					framework.K8sAPIs.Delete(ctx, mesh)
				}()
				assert.NoError(t, waitUntilStatusChanged(ctx, framework, mesh))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curMesh))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curMesh.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curMesh.Status, opts))
				}
			} else {
				// For all other test cases
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curMesh))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curMesh.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curMesh.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
			}
		})
		afterEach(framework)
	}
}

func waitUntilStatusChanged(ctx context.Context, framework *functional.Framework, mesh *servicemeshapi.Mesh) error {
	observedMesh := &servicemeshapi.Mesh{}
	key := namespace.NewNamespacedName(mesh)
	if err := framework.K8sClient.Get(ctx, key, observedMesh); err != nil {
		return err
	}
	oldStatus := commons.GetServiceMeshCondition(&observedMesh.Status, servicemeshapi.ServiceMeshActive).Status
	oldDependenciesStatus := commons.GetServiceMeshCondition(&observedMesh.Status, servicemeshapi.ServiceMeshDependenciesActive).Status
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedMesh); err != nil {
			return false, err
		}
		if observedMesh != nil && commons.GetServiceMeshCondition(&observedMesh.Status, servicemeshapi.ServiceMeshActive).Status != oldStatus {
			return true, nil
		}
		if observedMesh != nil && commons.GetServiceMeshCondition(&observedMesh.Status, servicemeshapi.ServiceMeshDependenciesActive).Status != oldDependenciesStatus {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, mesh *servicemeshapi.Mesh) error {
	observedMesh := &servicemeshapi.Mesh{}
	key := namespace.NewNamespacedName(mesh)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedMesh); err != nil {
			return false, err
		}
		if observedMesh != nil && commons.GetServiceMeshCondition(&observedMesh.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
