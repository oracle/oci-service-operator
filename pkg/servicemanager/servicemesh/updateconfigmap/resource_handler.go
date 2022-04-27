/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package updateconfigmap

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

const falseStr = "false"

// ResourceHandler handles polling the service mesh control plane to obtain proxy version details
type ResourceHandler interface {
	UpdateLatestProxyVersion(ctx context.Context, configMap *corev1.ConfigMap) error
	UpdateServiceMeshClientHost(configMap *corev1.ConfigMap)
}

type defaultResourceHandler struct {
	client            client.Client
	serviceMeshClient services.ServiceMeshClient
	log               loggerutil.OSOKLogger
}

func NewDefaultResourceHandler(
	client client.Client,
	serviceMeshClient services.ServiceMeshClient,
	log loggerutil.OSOKLogger) ResourceHandler {
	return &defaultResourceHandler{
		client:            client,
		serviceMeshClient: serviceMeshClient,
		log:               log,
	}
}

// UpdateServiceMeshClientHost updates service mesh control plane client host using the CP_ENDPOINT value in the service mesh configmap
func (h *defaultResourceHandler) UpdateServiceMeshClientHost(configMap *corev1.ConfigMap) {
	cpEndpoint, cpEndpointPresent := configMap.Data[commons.CpEndpointInMeshConfigMap]
	if cpEndpointPresent {
		h.serviceMeshClient.SetClientHost(cpEndpoint)
	}
}

// UpdateLatestProxyVersion updates Configmap with the latest proxy version obtained from the control plane
func (h *defaultResourceHandler) UpdateLatestProxyVersion(ctx context.Context, configMap *corev1.ConfigMap) error {
	oldProxyVersion, oldProxyVersionPresent := configMap.Data[commons.ProxyLabelInMeshConfigMap]
	updateProxyVersion, updateProxyVersionPresent := configMap.Data[commons.AutoUpdateProxyVersion]

	if updateProxyVersionPresent && strings.ToLower(updateProxyVersion) == falseStr && oldProxyVersionPresent {
		h.log.InfoLog("AutoUpdate is set to false. Keeping old version " + oldProxyVersion)
		return nil
	}

	res, err := h.serviceMeshClient.GetProxyDetails(ctx)
	if err != nil {
		return err
	}

	newProxyVersion := *res
	h.log.InfoLog("Configmap", "Latest Proxy version", newProxyVersion)

	// If proxy version is outdated, the config map is updated with the latest version
	if oldProxyVersion != newProxyVersion {
		newConfigMap := configMap.DeepCopy()
		if newConfigMap.Data == nil {
			newConfigMap.Data = make(map[string]string)
		}
		newConfigMap.Data[commons.ProxyLabelInMeshConfigMap] = newProxyVersion

		if err := h.client.Patch(ctx, newConfigMap, client.MergeFrom(configMap)); err != nil {
			h.log.ErrorLog(err, "Error in updating Configmap with new proxy version")
			return err
		}
		h.log.InfoLog("ConfigMap successfully updated with new proxy version")
	}

	return nil
}
