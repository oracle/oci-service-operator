/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceupdate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
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

// TODO: Add evict pod negative test case with PDB
func TestServiceUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	tests := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "update service and evict the pod successfully",
			expectedErr: errors.New("Pod \"my-pod\" not found"),
		},
	}

	for _, tt := range tests {
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
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
			namespace.Labels[commons.ProxyInjectionLabel] = commons.Enabled
			assert.NoError(t, framework.K8sAPIs.Create(ctx, namespace))

			// create a service with an unexpected selector
			service := functional.GetService()
			service.Spec.Selector["app"] = "my-service-1"
			assert.NoError(t, framework.K8sAPIs.Create(ctx, service))

			pod := functional.GetPod()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, pod))

			virtualDeploymentBinding := functional.GetApiVirtualDeploymentBinding()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeploymentBinding))

			// make sure the pod exists
			curPod := &corev1.Pod{}
			key := ns.NewNamespacedName(pod)
			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curPod))

			// update service with a correct selector
			oldService := service.DeepCopy()
			service.Spec.Selector["app"] = "my-service"
			assert.NoError(t, framework.K8sAPIs.Update(ctx, service, oldService))

			// If the pod is successfully evicted, the Get request should fail because of not found error
			err := framework.K8sAPIs.Get(ctx, key, curPod)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
		afterEach(framework)
	}
}
