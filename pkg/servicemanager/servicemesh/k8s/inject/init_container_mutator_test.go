/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

const (
	proxyImageValue  = "sm-proxy-image-v1.17.0.1"
	mdsEndpointValue = "http://144.25.97.129:443"
)

func Test_init_container_mutate(t *testing.T) {
	tests := []struct {
		name      string
		configmap *corev1.ConfigMap
		pod       *corev1.Pod
		want      *corev1.Pod
	}{
		{
			name: "Test init container mutator with initContainerImage supplied ",
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
							Name:  commons.InitContainerName,
							Image: proxyImageValue,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{commons.AllCapabilites},
									Add:  []corev1.Capability{commons.NetAdminCapability, commons.NetRawCapability},
								},
								Privileged:               conversions.Bool(false),
								RunAsUser:                conversions.Int64(0),
								RunAsGroup:               conversions.Int64(0),
								RunAsNonRoot:             conversions.Bool(false),
								ReadOnlyRootFilesystem:   conversions.Bool(true),
								AllowPrivilegeEscalation: conversions.Bool(false),
							},
							Env: []corev1.EnvVar{
								{
									Name:  string(commons.ConfigureIpTablesEnvName),
									Value: string(commons.ConfigureIpTablesEnvValue),
								},
								{
									Name:  string(commons.EnvoyPortEnvVarName),
									Value: string(commons.EnvoyPortEnvVarValue),
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
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		m := newInitMutator(tt.configmap.Data[commons.ProxyLabelInMeshConfigMap])
		err := m.mutate(tt.pod)
		assert.NoError(t, err)
		opts := equality.IgnoreFakeClientPopulatedFields()
		assert.True(t, cmp.Equal(tt.want, tt.pod, opts), "diff", cmp.Diff(tt.want, tt.pod, opts))
	}
}
