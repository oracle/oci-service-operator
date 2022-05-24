/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualservice

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
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
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

func TestVirtualService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		compartmentId            string
		freeformTags             map[string]string
		definedTagsSdk           map[string]map[string]interface{}
		definedTags              map[string]api.MapValue
		virtualDeployment        *servicemeshapi.VirtualDeployment
		virtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable
		ingressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable
		accessPolicy             *servicemeshapi.AccessPolicy
		cpError                  error
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create virtualService",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
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
			name: "change virtualService compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
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
			name: "update virtualService freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
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
			name: "update virtualService definedTags",
			args: args{
				definedTagsSdk: map[string]map[string]interface{}{
					"definedTag2": {"foo2": "bar2"},
				},
				definedTags: map[string]api.MapValue{
					"definedTag2": {"foo2": "bar2"},
				},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
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
			name: "delete virtualService failed because of associated virtualDeployment",
			args: args{
				virtualDeployment: functional.GetApiVirtualDeployment(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual service when there are virtual deployment resources associated",
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
			name: "delete virtualService failed because of associated virtualServiceRouteTables",
			args: args{
				virtualServiceRouteTable: functional.GetApiVirtualServiceRouteTable(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual service when there are virtual service route table resources associated",
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
			name: "delete virtualService failed because of associated ingressGatewayRouteTable",
			args: args{
				ingressGatewayRouteTable: functional.GetApiIngressGatewayRouteTable(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual service when there are ingress gateway route table resources associated",
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
			name: "delete virtualService failed because of associated accessPolicy",
			args: args{
				accessPolicy: functional.GetApiAccessPolicy(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete virtual service when there are access policy resources associated",
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
			name: "delete virtualService failed because of associated dependencies in control plane",
			args: args{
				cpError: errors.NewServiceError(409, "Conflict", "IncorrectState. Can't delete the resource with dependencies", "123"),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				VirtualServiceId: api.OCID("my-virtualservice-id"),
				VirtualServiceMtls: &servicemeshapi.VirtualServiceMutualTransportLayerSecurity{
					Mode:          servicemeshapi.MutualTransportLayerSecurityModeDisabled,
					CertificateId: conversions.OCID("certificate-authority-id"),
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
			framework.MeshClient.EXPECT().DeleteMesh(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			assert.NoError(t, framework.K8sAPIs.Create(ctx, mesh))

			// Create the virtualService
			virtualService := functional.GetApiVirtualService()
			framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
			if tt.args.cpError != nil {
				framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(tt.args.cpError).AnyTimes()
			} else {
				framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}

			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkVirtualService0 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService1 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateUpdating)
				sdkVirtualService1.CompartmentId = &tt.args.compartmentId
				sdkVirtualService2 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService2.CompartmentId = &tt.args.compartmentId
				sdkVirtualService3 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateUpdating)
				sdkVirtualService3.CompartmentId = &tt.args.compartmentId
				sdkVirtualService4 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService2, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService3, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeVirtualServiceCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateVirtualService(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualService := virtualService.DeepCopy()
				virtualService.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, virtualService, oldVirtualService)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualService))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkVirtualService0 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService1 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateUpdating)
				sdkVirtualService1.FreeformTags = tt.args.freeformTags
				sdkVirtualService2 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualService(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualService := virtualService.DeepCopy()
				virtualService.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, virtualService, oldVirtualService)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualService))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkVirtualService0 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService1 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateUpdating)
				sdkVirtualService1.DefinedTags = tt.args.definedTagsSdk
				sdkVirtualService2 := functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive)
				sdkVirtualService2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService0, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService1, nil).Times(1),
					framework.MeshClient.EXPECT().GetVirtualService(gomock.Any(), gomock.Any()).Return(sdkVirtualService2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateVirtualService(gomock.Any(), gomock.Any()).Return(nil)
				oldVirtualService := virtualService.DeepCopy()
				virtualService.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, virtualService, oldVirtualService)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, virtualService))
			}

			curVirtualService := &servicemeshapi.VirtualService{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(virtualService)

			if tt.args.virtualDeployment != nil {
				virtualDeployment := tt.args.virtualDeployment.DeepCopy()
				// For virtualService deletion fails with associated virtualDeployment test case, create virtualDeployment and validate the deletion
				framework.MeshClient.EXPECT().CreateVirtualDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeployment))

				go func() {
					framework.K8sAPIs.Delete(ctx, virtualService)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualService))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeployment))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualService)
			} else if tt.args.virtualServiceRouteTable != nil {
				// For virtualService deletion fails with associated virtualServiceRouteTable test case, create virtualServiceRouteTable and validate the deletion
				virtualDeployment := functional.GetApiVirtualDeployment()
				framework.MeshClient.EXPECT().CreateVirtualDeployment(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualDeployment(sdk.VirtualDeploymentLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualDeployment(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeployment))

				virtualServiceRouteTable := tt.args.virtualServiceRouteTable.DeepCopy()
				framework.MeshClient.EXPECT().CreateVirtualServiceRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualServiceRouteTable(sdk.VirtualServiceRouteTableLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualServiceRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualServiceRouteTable))

				go func() {
					framework.K8sAPIs.Delete(ctx, virtualService)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualService))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualServiceRouteTable))
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeployment))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualService)
			} else if tt.args.ingressGatewayRouteTable != nil {
				// For virtualService deletion fails with associated ingressGatewayRouteTable test case, create ingressGatewayRouteTable and validate the deletion
				ingressGateway := functional.GetApiIngressGateway()
				framework.MeshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGateway))

				ingressGatewayRouteTable := tt.args.ingressGatewayRouteTable.DeepCopy()
				framework.MeshClient.EXPECT().CreateIngressGatewayRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayRouteTable))

				go func() {
					framework.K8sAPIs.Delete(ctx, virtualService)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualService))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGatewayRouteTable))
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGateway))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualService)
			} else if tt.args.accessPolicy != nil {
				// For virtualService deletion fails with associated accessPolicy test case, create accessPolicy and validate the deletion
				accessPolicy := tt.args.accessPolicy.DeepCopy()
				framework.MeshClient.EXPECT().CreateAccessPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkAccessPolicy(sdk.AccessPolicyLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteAccessPolicy(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, accessPolicy))
				go func() {
					framework.K8sAPIs.Delete(ctx, virtualService)
				}()
				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualService))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, accessPolicy))
				framework.K8sAPIs.WaitUntilDeleted(ctx, virtualService)
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
			} else if tt.args.cpError != nil {
				go func() {
					framework.K8sAPIs.Delete(ctx, virtualService)
				}()
				assert.NoError(t, waitUntilStatusChanged(ctx, framework, virtualService))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}
			} else {
				// For all other test cases
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualService))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualService.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualService.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
			}
		})
		afterEach(framework)
	}
}

func waitUntilStatusChanged(ctx context.Context, framework *functional.Framework, virtualService *servicemeshapi.VirtualService) error {
	observedVirtualService := &servicemeshapi.VirtualService{}
	key := namespace.NewNamespacedName(virtualService)
	if err := framework.K8sClient.Get(ctx, key, observedVirtualService); err != nil {
		return err
	}
	oldStatus := commons.GetServiceMeshCondition(&observedVirtualService.Status, servicemeshapi.ServiceMeshActive).Status
	oldDependenciesStatus := commons.GetServiceMeshCondition(&observedVirtualService.Status, servicemeshapi.ServiceMeshDependenciesActive).Status
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedVirtualService); err != nil {
			return false, err
		}
		if observedVirtualService != nil && commons.GetServiceMeshCondition(&observedVirtualService.Status, servicemeshapi.ServiceMeshActive).Status != oldStatus {
			return true, nil
		}
		if observedVirtualService != nil && commons.GetServiceMeshCondition(&observedVirtualService.Status, servicemeshapi.ServiceMeshDependenciesActive).Status != oldDependenciesStatus {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, virtualService *servicemeshapi.VirtualService) error {
	observedVirtualService := &servicemeshapi.VirtualService{}
	key := namespace.NewNamespacedName(virtualService)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedVirtualService); err != nil {
			return false, err
		}
		if observedVirtualService != nil && commons.GetServiceMeshCondition(&observedVirtualService.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
