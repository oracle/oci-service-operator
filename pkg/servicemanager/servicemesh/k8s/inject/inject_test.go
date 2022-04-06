/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

func Test_inject(t *testing.T) {
	tests := []struct {
		name      string
		configmap *corev1.ConfigMap
		pod       *corev1.Pod
		want      *corev1.Pod
		wantError string
	}{
		{
			name: "Test inject with sidecarImage supplied",
			configmap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName,
				map[string]string{commons.ProxyLabelInMeshConfigMap: proxyImageValue,
					commons.MdsEndpointInMeshConfigMap: mdsEndpointValue}),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "product-v1-binding",
					Namespace: "product",
					Labels: map[string]string{
						"app": "name",
					},
				},
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "product-v1-binding",
					Namespace: "product",
					Labels: map[string]string{
						"app": "name",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            commons.InitContainerName,
							Image:           proxyImageValue,
							SecurityContext: &corev1.SecurityContext{Capabilities: &corev1.Capabilities{Add: []corev1.Capability{commons.NetAdminCapability}}},
							Env: []corev1.EnvVar{
								{
									Name:  string(commons.ConfigureIpTablesEnvName),
									Value: string(commons.ConfigureIpTablesEnvValue),
								},
								{
									Name:  string(commons.EnvoyPortEnvVarName),
									Value: string(commons.EnvoyPortEnvVarValue),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  commons.ProxyContainerName,
							Image: proxyImageValue,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: commons.StatsPort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: string(commons.DeploymentId),
								},
								{
									Name:  string(commons.ProxyLogLevel),
									Value: string(commons.ProxyLogLevelError),
								},
								{
									Name: string(commons.IPAddress),
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  commons.MdsEndpointInMeshConfigMap,
									Value: mdsEndpointValue,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPURequestSize)),
									corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryRequestSize)),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPULimitSize)),
									corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryLimitSize)),
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: commons.LivenessProbeEndpointPath,
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: commons.LivenessProbeEndpointPort,
										},
									},
								},
								InitialDelaySeconds: commons.LivenessProbeInitialDelaySeconds,
							},
						},
					},
				},
			},
		},
		{
			name: "Test inject with no sidecarImage",
			configmap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName,
				map[string]string{commons.MdsEndpointInMeshConfigMap: mdsEndpointValue}),
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "product-v1-binding",
					Namespace: "product",
					Labels: map[string]string{
						"app": "name",
					},
				},
			},
			wantError: string(commons.NoSidecarImageFound),
		},
	}
	for _, tt := range tests {
		vdb := NewVdbWithVdRef("product-v1-binding", "product", "product", "test")
		m := NewMutateHandler(tt.configmap, *vdb)
		err := m.Inject(tt.pod)
		if tt.want != nil {
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.NoError(t, err)
			assert.True(t, cmp.Equal(tt.want, tt.pod, opts), "diff", cmp.Diff(tt.want, tt.pod, opts))
		} else {
			assert.Error(t, err, tt.wantError)
		}
	}
}
