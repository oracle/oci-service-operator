/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

type proxyMutator struct {
	envoyImage  string
	mdsEndpoint string
	vdb         servicemeshapi.VirtualDeploymentBinding
}

func newEnvoyMutator(envoyImage string,
	mdsEndpoint string,
	vdb servicemeshapi.VirtualDeploymentBinding) *proxyMutator {
	return &proxyMutator{
		envoyImage:  envoyImage,
		mdsEndpoint: mdsEndpoint,
		vdb:         vdb,
	}
}

func (mctx *proxyMutator) mutate(pod *corev1.Pod) error {

	annotations := pod.Annotations
	proxyLogLevel, ok := annotations[commons.ProxyLogLevelAnnotation]
	if !ok {
		proxyLogLevel = string(commons.DefaultProxyLogLevel)
	}
	// Validate the log level present
	if !(commons.IsStringPresent(commons.ProxyLogLevels, strings.ToLower(proxyLogLevel))) {
		return errors.New(string(commons.InValidProxyLogAnnotation))
	}

	resourceRequests := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPURequestSize)),
			corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryRequestSize)),
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse(string(commons.SidecarCPULimitSize)),
			corev1.ResourceMemory: resource.MustParse(string(commons.SidecarMemoryLimitSize)),
		},
	}

	if mctx.vdb.Spec.Resources != nil {
		resourceRequests = *mctx.vdb.Spec.Resources
	}

	envoy := corev1.Container{
		Name:  commons.ProxyContainerName,
		Image: mctx.envoyImage,
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: commons.StatsPort,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  string(commons.DeploymentId),
				Value: string(mctx.vdb.Status.VirtualDeploymentId),
			},
			{
				Name:  string(commons.ProxyLogLevel),
				Value: strings.ToLower(proxyLogLevel),
			},
			{
				Name: string(commons.IPAddress),
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
		},
		Resources: resourceRequests,

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
	}
	if mctx.mdsEndpoint != "" {
		envoy.Env = append(envoy.Env, corev1.EnvVar{Name: commons.MdsEndpointInMeshConfigMap, Value: mctx.mdsEndpoint})
	}

	pod.Spec.Containers = append(pod.Spec.Containers, envoy)
	return nil
}
