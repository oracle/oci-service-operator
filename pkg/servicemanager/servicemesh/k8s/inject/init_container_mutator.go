/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

type initMutator struct {
	initContainerImage string
}

type InitContainerEnvVars string

func (mctx *initMutator) mutate(pod *corev1.Pod) error {
	initC := corev1.Container{
		Name:            commons.InitContainerName,
		Image:           mctx.initContainerImage,
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
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initC)
	return nil
}

func newInitMutator(initContainerImage string) *initMutator {
	return &initMutator{
		initContainerImage: initContainerImage,
	}
}
