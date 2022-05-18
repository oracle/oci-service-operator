/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package inject

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

type MutateHandler struct {
	config *corev1.ConfigMap
	vdb    servicemeshapi.VirtualDeploymentBinding
}

func NewMutateHandler(config *corev1.ConfigMap,
	vdb servicemeshapi.VirtualDeploymentBinding) *MutateHandler {
	return &MutateHandler{
		config: config,
		vdb:    vdb}
}

type PodMutator interface {
	mutate(*corev1.Pod) error
}

func (h *MutateHandler) Inject(pod *corev1.Pod) error {
	var logger = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("Mutator").WithName("Inject")}

	sidecarImage, sidecarOk := h.config.Data[commons.ProxyLabelInMeshConfigMap]

	if !sidecarOk {
		err := errors.New(string(commons.NoSidecarImageFound))
		logger.ErrorLogWithFixedMessage(nil, err, string(commons.NoSidecarImageFound))
		return err
	}

	mdsEndpoint := h.config.Data[commons.MdsEndpointInMeshConfigMap]
	mutators := []PodMutator{
		newEnvoyMutator(sidecarImage, mdsEndpoint, h.vdb),
		newInitMutator(sidecarImage),
	}

	for _, mutator := range mutators {
		logger.InfoLogWithFixedMessage(nil, "Attempting to mutate", "vd", (string)(h.vdb.Status.VirtualDeploymentId), "vs",
			(string)(h.vdb.Status.VirtualServiceId))
		err := mutator.mutate(pod)
		if err != nil {
			logger.ErrorLogWithFixedMessage(nil, err, "failed to mutate the pod")
			return err
		}
	}
	return nil
}
