/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	ns "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
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

func TestVirtualDeploymentBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	type args struct {
		namespaceEnabled    bool
		hasResourceRequests bool
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus *servicemeshapi.ServiceMeshStatus
		expectedErr    error
	}{
		{
			name: "create virtualDeploymentBinding with namespace enabled",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:                api.OCID("my-mesh-id"),
				VirtualServiceId:      api.OCID("my-virtualservice-id"),
				VirtualDeploymentId:   api.OCID("my-virtualdeployment-id"),
				VirtualServiceName:    servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentName: servicemeshapi.Name("my-virtualdeployment"),
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
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			args: args{
				namespaceEnabled: true,
			},
			expectedErr: errors.New("pods \"my-pod\" not found"),
		},
		{
			name: "create virtualDeploymentBinding with namespace enabled and has resource requests enabled",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:                api.OCID("my-mesh-id"),
				VirtualServiceId:      api.OCID("my-virtualservice-id"),
				VirtualDeploymentId:   api.OCID("my-virtualdeployment-id"),
				VirtualServiceName:    servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentName: servicemeshapi.Name("my-virtualdeployment"),
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
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			args: args{
				namespaceEnabled:    true,
				hasResourceRequests: true,
			},
			expectedErr: errors.New("pods \"my-pod\" not found"),
		},
		{
			name: "create virtualDeploymentBinding with namespace disabled",
			expectedStatus: &servicemeshapi.ServiceMeshStatus{
				MeshId:                api.OCID("my-mesh-id"),
				VirtualServiceId:      api.OCID("my-virtualservice-id"),
				VirtualDeploymentId:   api.OCID("my-virtualdeployment-id"),
				VirtualServiceName:    servicemeshapi.Name("my-virtualservice"),
				VirtualDeploymentName: servicemeshapi.Name("my-virtualdeployment"),
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
						Type: servicemeshapi.ServiceMeshActive,
						ResourceCondition: servicemeshapi.ResourceCondition{
							Status:             metav1.ConditionTrue,
							Reason:             string(commons.Successful),
							Message:            string(commons.ResourceActiveVDB),
							ObservedGeneration: 1,
						},
					},
				},
			},
			args:        args{},
			expectedErr: errors.New("pods \"my-pod\" not found"),
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

			namespace := functional.GetSidecarInjectNamespace()
			if tt.args.namespaceEnabled {
				namespace.Labels[commons.ProxyInjectionLabel] = commons.Enabled
			}

			assert.NoError(t, framework.K8sAPIs.Create(ctx, namespace))

			service := functional.GetService()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, service))

			pod := functional.GetPod()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, pod))

			virtualDeploymentBinding := functional.GetApiVirtualDeploymentBinding()

			if tt.args.hasResourceRequests {
				virtualDeploymentBinding.Spec.Resources = &corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPURequestSize)),
						corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryRequestSize)),
					},
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPULimitSize)),
						corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryLimitSize)),
					},
				}

			}
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeploymentBinding))

			curVirtualDeploymentBinding := &servicemeshapi.VirtualDeploymentBinding{}
			opts := equality.IgnoreFakeClientPopulatedFields()
			key := ns.NewNamespacedName(virtualDeploymentBinding)
			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curVirtualDeploymentBinding))
			if tt.expectedStatus != nil {
				assert.True(t, cmp.Equal(*tt.expectedStatus, curVirtualDeploymentBinding.Status, opts), "diff", cmp.Diff(*tt.expectedStatus, curVirtualDeploymentBinding.Status, opts))
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeploymentBinding))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, service))

			// If the pod is successfully evicted, it should failed the deletion because of no found error
			err := framework.K8sAPIs.Delete(ctx, pod)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualDeployment))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, virtualService))
			assert.NoError(t, framework.K8sAPIs.Delete(ctx, mesh))
		})
		afterEach(framework)
	}
}
