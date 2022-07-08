/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

func Test_proxy_container_mutate(t *testing.T) {
	tests := []struct {
		name      string
		configmap *corev1.ConfigMap
		pod       *corev1.Pod
		want      *corev1.Pod
		vdId      string
	}{
		{
			name: "Pod with mds endpoint provided in configmap",
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
					Containers: []corev1.Container{
						{
							Name:  commons.ProxyContainerName,
							Image: proxyImageValue,
							SecurityContext: &corev1.SecurityContext{
								Privileged:             conversions.Bool(false),
								RunAsUser:              conversions.Int64(0),
								RunAsGroup:             conversions.Int64(0),
								RunAsNonRoot:           conversions.Bool(false),
								ReadOnlyRootFilesystem: conversions.Bool(false),
							},
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
									Name: string(commons.PodIp),
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
			name: "Pod without mds endpoint in configmap",
			configmap: NewConfigMap(commons.OsokNamespace, commons.MeshConfigMapName,
				map[string]string{commons.ProxyLabelInMeshConfigMap: proxyImageValue}),
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
					Containers: []corev1.Container{
						{
							Name:  commons.ProxyContainerName,
							Image: proxyImageValue,
							SecurityContext: &corev1.SecurityContext{
								Privileged:             conversions.Bool(false),
								RunAsUser:              conversions.Int64(0),
								RunAsGroup:             conversions.Int64(0),
								RunAsNonRoot:           conversions.Bool(false),
								ReadOnlyRootFilesystem: conversions.Bool(false),
							},
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
									Name: string(commons.PodIp),
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
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
			name: "Pod with different log level",
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
					Annotations: map[string]string{
						commons.ProxyLogLevelAnnotation: string(commons.ProxyLogLevelInfo),
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
					Annotations: map[string]string{
						commons.ProxyLogLevelAnnotation: string(commons.ProxyLogLevelInfo),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  commons.ProxyContainerName,
							Image: proxyImageValue,
							SecurityContext: &corev1.SecurityContext{
								Privileged:             conversions.Bool(false),
								RunAsUser:              conversions.Int64(0),
								RunAsGroup:             conversions.Int64(0),
								RunAsNonRoot:           conversions.Bool(false),
								ReadOnlyRootFilesystem: conversions.Bool(false),
							},
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
									Value: string(commons.ProxyLogLevelInfo),
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
									Name: string(commons.PodIp),
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
			name: "Populate container env variables",
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
					Annotations: map[string]string{
						commons.ProxyLogLevelAnnotation: string(commons.ProxyLogLevelInfo),
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
					Annotations: map[string]string{
						commons.ProxyLogLevelAnnotation: string(commons.ProxyLogLevelInfo),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  commons.ProxyContainerName,
							Image: proxyImageValue,
							SecurityContext: &corev1.SecurityContext{
								Privileged:             conversions.Bool(false),
								RunAsUser:              conversions.Int64(0),
								RunAsGroup:             conversions.Int64(0),
								RunAsNonRoot:           conversions.Bool(false),
								ReadOnlyRootFilesystem: conversions.Bool(false),
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: commons.StatsPort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  string(commons.DeploymentId),
									Value: "deploymentId",
								},
								{
									Name:  string(commons.ProxyLogLevel),
									Value: string(commons.ProxyLogLevelInfo),
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
									Name: string(commons.PodIp),
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
			vdId: "deploymentId",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdb := NewVdbWithVdRef("product-v1-binding", "product", "product", "test")
			if len(tt.vdId) != 0 {
				vdb.Status.VirtualDeploymentId = api.OCID(tt.vdId)
			}
			m := newEnvoyMutator(tt.configmap.Data[commons.ProxyLabelInMeshConfigMap], tt.configmap.Data[commons.MdsEndpointInMeshConfigMap], *vdb)
			err := m.mutate(tt.pod)
			assert.NoError(t, err)
			opts := equality.IgnoreFakeClientPopulatedFields()
			assert.True(t, cmp.Equal(tt.want, tt.pod, opts), "diff", cmp.Diff(tt.want, tt.pod, opts))
		})
	}
}
