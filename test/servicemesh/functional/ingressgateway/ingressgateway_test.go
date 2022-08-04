/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgateway

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

func TestIngressGateway(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		compartmentId            string
		freeformTags             map[string]string
		definedTagsSdk           map[string]map[string]interface{}
		definedTags              map[string]api.MapValue
		ingressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable
		virtualService           *servicemeshapi.VirtualService
		cpError                  error
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create ingressGateway",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
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
			name: "change ingressGateway compartmentId",
			args: args{
				compartmentId: "compartment-id-2",
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
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
			name: "update ingressGateway freeformTags",
			args: args{
				freeformTags: map[string]string{"freeformTag2": "value2"},
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
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
			name: "update ingressGateway definedTags",
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
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
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
			name: "delete ingressGateway failed because of associated ingressGatewayRouteTable",
			args: args{
				virtualService:           functional.GetApiVirtualService(),
				ingressGatewayRouteTable: functional.GetApiIngressGatewayRouteTable(),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
				},
				Conditions: []servicemeshapi.ServiceMeshCondition{
					{
						Type: servicemeshapi.ServiceMeshDependenciesActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionUnknown,
							Reason:             string(commons.DependenciesNotResolved),
							Message:            "cannot delete ingress gateway when there are ingress gateway route table resources associated",
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

			name: "delete ingressGateway failed because of associated dependencies in control plane",
			args: args{
				cpError: errors.NewServiceError(409, "Conflict", "IncorrectState. Can't delete the resource with dependencies", "123"),
			},
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:           api.OCID("my-mesh-id"),
				IngressGatewayId: api.OCID("my-ingressgateway-id"),
				IngressGatewayMtls: &servicemeshapi.IngressGatewayMutualTransportLayerSecurity{
					CertificateId: api.OCID("certificate-authority-id"),
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

			// Create the ingressGateway
			ingressGateway := functional.GetApiIngressGateway()
			framework.MeshClient.EXPECT().CreateIngressGateway(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive), nil).AnyTimes()
			if tt.args.cpError != nil {
				framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(tt.args.cpError).AnyTimes()
			} else {
				framework.MeshClient.EXPECT().DeleteIngressGateway(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			}
			assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGateway))

			// Change compartmentId
			if len(tt.args.compartmentId) > 0 {
				sdkIngressGateway0 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway1 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateUpdating)
				sdkIngressGateway1.CompartmentId = &tt.args.compartmentId
				sdkIngressGateway2 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway2.CompartmentId = &tt.args.compartmentId
				sdkIngressGateway3 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateUpdating)
				sdkIngressGateway3.CompartmentId = &tt.args.compartmentId
				sdkIngressGateway4 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway4.CompartmentId = &tt.args.compartmentId
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway2, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway3, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway4, nil).Times(1),
				)
				framework.MeshClient.EXPECT().ChangeIngressGatewayCompartment(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				framework.MeshClient.EXPECT().UpdateIngressGateway(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGateway := ingressGateway.DeepCopy()
				ingressGateway.Spec.CompartmentId = api.OCID(tt.args.compartmentId)
				err := framework.K8sAPIs.Update(ctx, ingressGateway, oldIngressGateway)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, ingressGateway))
			}

			// Update freeformTags
			if tt.args.freeformTags != nil {
				sdkIngressGateway0 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway1 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateUpdating)
				sdkIngressGateway1.FreeformTags = tt.args.freeformTags
				sdkIngressGateway2 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway2.FreeformTags = tt.args.freeformTags
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateIngressGateway(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGateway := ingressGateway.DeepCopy()
				ingressGateway.Spec.FreeFormTags = tt.args.freeformTags
				err := framework.K8sAPIs.Update(ctx, ingressGateway, oldIngressGateway)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, ingressGateway))
			}

			// Update definedTags
			if tt.args.definedTags != nil {
				sdkIngressGateway0 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway1 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateUpdating)
				sdkIngressGateway1.DefinedTags = tt.args.definedTagsSdk
				sdkIngressGateway2 := functional.GetSdkIngressGateway(sdk.IngressGatewayLifecycleStateActive)
				sdkIngressGateway2.DefinedTags = tt.args.definedTagsSdk
				gomock.InOrder(
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway0, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway1, nil).Times(1),
					framework.MeshClient.EXPECT().GetIngressGateway(gomock.Any(), gomock.Any()).Return(sdkIngressGateway2, nil).Times(1),
				)
				framework.MeshClient.EXPECT().UpdateIngressGateway(gomock.Any(), gomock.Any()).Return(nil)
				oldIngressGateway := ingressGateway.DeepCopy()
				ingressGateway.Spec.DefinedTags = tt.args.definedTags
				err := framework.K8sAPIs.Update(ctx, ingressGateway, oldIngressGateway)
				assert.NoError(t, err)
				assert.NoError(t, waitUntilSettled(ctx, framework, ingressGateway))
			}

			curIngressGateway := &servicemeshapi.IngressGateway{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := namespace.NewNamespacedName(ingressGateway)

			if tt.args.ingressGatewayRouteTable != nil && tt.args.virtualService != nil {
				// For ingressGateway deletion fails with associated ingressGatewayRouteTable test case, create ingressGatewayRouteTable and validate the deletion
				virtualService := tt.args.virtualService.DeepCopy()
				framework.MeshClient.EXPECT().CreateVirtualService(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkVirtualService(sdk.VirtualServiceLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteVirtualService(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualService))

				ingressGatewayRouteTable := tt.args.ingressGatewayRouteTable.DeepCopy()
				framework.MeshClient.EXPECT().CreateIngressGatewayRouteTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(functional.GetSdkIngressGatewayRouteTable(sdk.IngressGatewayRouteTableLifecycleStateActive), nil).AnyTimes()
				framework.MeshClient.EXPECT().DeleteIngressGatewayRouteTable(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				assert.NoError(t, framework.K8sAPIs.Create(ctx, ingressGatewayRouteTable))

				go func() {
					framework.K8sAPIs.Delete(ctx, ingressGateway)
				}()

				assert.NoError(t, waitUntilStatusChanged(ctx, framework, ingressGateway))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGateway))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGateway.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGateway.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGatewayRouteTable))
				assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
				framework.K8sAPIs.WaitUntilDeleted(ctx, ingressGateway)
			} else if tt.args.cpError != nil {
				go func() {
					framework.K8sAPIs.Delete(ctx, ingressGateway)
				}()
				assert.NoError(t, waitUntilStatusChanged(ctx, framework, ingressGateway))
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGateway))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGateway.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGateway.Status, opts))
				}
			} else {
				// For all other test cases
				assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curIngressGateway))
				if tt.expectedStatus != nil {
					assert.True(t, cmp.Equal(*tt.expectedStatus, curIngressGateway.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curIngressGateway.Status, opts))
				}

				assert.NoError(t, framework.K8sAPIs.Delete(ctx, ingressGateway))
			}
		})
		afterEach(framework)
	}
}

func waitUntilStatusChanged(ctx context.Context, framework *functional.Framework, ingressGateway *servicemeshapi.IngressGateway) error {
	observedIngressGateway := &servicemeshapi.IngressGateway{}
	key := namespace.NewNamespacedName(ingressGateway)
	if err := framework.K8sClient.Get(ctx, key, observedIngressGateway); err != nil {
		return err
	}
	oldStatus := commons.GetServiceMeshCondition(&observedIngressGateway.Status, servicemeshapi.ServiceMeshActive).Status
	oldDependenciesStatus := commons.GetServiceMeshCondition(&observedIngressGateway.Status, servicemeshapi.ServiceMeshDependenciesActive).Status
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedIngressGateway); err != nil {
			return false, err
		}
		if observedIngressGateway != nil && commons.GetServiceMeshCondition(&observedIngressGateway.Status, servicemeshapi.ServiceMeshActive).Status != oldStatus {
			return true, nil
		}
		if observedIngressGateway != nil && commons.GetServiceMeshCondition(&observedIngressGateway.Status, servicemeshapi.ServiceMeshDependenciesActive).Status != oldDependenciesStatus {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}

func waitUntilSettled(ctx context.Context, framework *functional.Framework, ingressGateway *servicemeshapi.IngressGateway) error {
	observedIngressGateway := &servicemeshapi.IngressGateway{}
	key := namespace.NewNamespacedName(ingressGateway)
	return wait.PollImmediateUntil(commons.PollInterval, func() (bool, error) {
		if err := framework.K8sClient.Get(ctx, key, observedIngressGateway); err != nil {
			return false, err
		}
		if observedIngressGateway != nil && commons.GetServiceMeshCondition(&observedIngressGateway.Status, servicemeshapi.ServiceMeshActive).Status == metav1.ConditionTrue {
			return true, nil
		}
		return false, nil
	}, ctx.Done())
}
