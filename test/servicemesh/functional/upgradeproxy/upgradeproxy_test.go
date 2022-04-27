/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package upgradeproxy

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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	ns "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	"github.com/oracle/oci-service-operator/test/servicemesh/functional"
	"github.com/oracle/oci-service-operator/test/servicemesh/k8s"
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
func TestUpgradeProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Service Mesh Functional Tests during build.")
	}
	tests := []struct {
		name                string
		expectedErr         error
		expectedAnnotations map[string]string
		addAnnotation       bool
	}{
		{
			name:          "evict pod successfully",
			expectedErr:   errors.New("Pod \"my-pod\" not found"),
			addAnnotation: true,
		},
		{
			name: "add annotation successfully",
			expectedAnnotations: map[string]string{
				commons.OutdatedProxyAnnotation: "true",
			},
			addAnnotation: false,
		},
	}

	for _, tt := range tests {
		framework := beforeEach(t)
		time.Sleep(2 * time.Second)
		t.Run(tt.name, func(t *testing.T) {
			// Create config map
			framework.CreateNamespace(ctx, commons.OsokNamespace)
			configmap := functional.GetConfigMap()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, configmap))

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

			service := functional.GetService()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, service))

			virtualDeploymentBinding := functional.GetApiVirtualDeploymentBinding()
			assert.NoError(t, framework.K8sAPIs.Create(ctx, virtualDeploymentBinding))

			// create a pod contains the side car proxy
			pod := k8s.NewPodWithServiceMeshProxy("my-pod", "sidecar-inject-namespace")
			if tt.addAnnotation {
				pod.ObjectMeta.Annotations = make(map[string]string)
				pod.ObjectMeta.Annotations[commons.VirtualDeploymentBindingAnnotation] = client.ObjectKeyFromObject(virtualDeploymentBinding).String()
			}
			assert.NoError(t, framework.K8sAPIs.Create(ctx, pod))

			// make sure the pod exists
			curPod := &corev1.Pod{}
			key := ns.NewNamespacedName(pod)
			assert.NoError(t, framework.K8sAPIs.Get(ctx, key, curPod))

			// Update config map with a newer version of sm-proxy-image
			oldConfigmap := configmap.DeepCopy()
			configmap.Data[commons.ProxyLabelInMeshConfigMap] = "sm-proxy-image:0.0.2"
			assert.NoError(t, framework.K8sAPIs.Update(ctx, configmap, oldConfigmap))

			err := framework.K8sAPIs.Get(ctx, key, curPod)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.True(t, cmp.Equal(tt.expectedAnnotations, curPod.ObjectMeta.Annotations), "diff", cmp.Diff(tt.expectedAnnotations, curPod.ObjectMeta.Annotations))
			}
		})
		afterEach(framework)
	}
}
