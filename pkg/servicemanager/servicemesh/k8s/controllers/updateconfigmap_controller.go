/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/updateconfigmap"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

// UpdateConfigMapController helps in updating the ConfigMap with latest proxy details
type UpdateConfigMapController struct {
	client    client.Client
	handler   updateconfigmap.ResourceHandler
	log       loggerutil.OSOKLogger
	namespace string
}

func NewUpdateConfigMapController(
	client client.Client,
	handler updateconfigmap.ResourceHandler,
	log loggerutil.OSOKLogger,
	namespace string) *UpdateConfigMapController {
	return &UpdateConfigMapController{client: client, log: log, handler: handler, namespace: namespace}
}

// UpdateClientHostEndpoints polls the global configmap and updates the CP and MDS client hosts if endpoints have changed
func (c *UpdateConfigMapController) UpdateClientHostEndpoints(ctx context.Context) {
	serviceMeshConfigMap := &corev1.ConfigMap{}
	err := c.client.Get(ctx, types.NamespacedName{Namespace: c.namespace, Name: commons.MeshConfigMapName}, serviceMeshConfigMap)
	if err != nil {
		c.log.ErrorLog(err, "Error while fetching the configmap")
		return
	}
	c.handler.UpdateServiceMeshClientHost(serviceMeshConfigMap)
}

// PollServiceMeshProxyDetailEndpoint polls the proxy detail endpoint via the resource handler
func (c *UpdateConfigMapController) PollServiceMeshProxyDetailEndpoint(ctx context.Context) error {
	serviceMeshConfigMap := &corev1.ConfigMap{}
	err := c.client.Get(ctx, types.NamespacedName{Namespace: c.namespace, Name: commons.MeshConfigMapName}, serviceMeshConfigMap)
	if err != nil {
		c.log.ErrorLog(err, "Error while fetching the configmap")
		return err
	}
	c.log.InfoLog("Configmap", "Current proxy version", serviceMeshConfigMap.Data[commons.ProxyLabelInMeshConfigMap])

	err = c.handler.UpdateLatestProxyVersion(ctx, serviceMeshConfigMap)
	if err != nil {
		c.log.ErrorLog(err, "Error in updating Configmap with new proxy version")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
// Triggers a function to poll the mesh control plane every ten minutes to check for the latest proxy version
// Additionally trigger functions to check service mesh configmap for updated CP endpoints and set them in the clients
func (c *UpdateConfigMapController) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		ticker := time.NewTicker(commons.PollControlPlaneEndpointInterval)
		initialize := make(chan bool, 1)
		initialize <- true
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = c.PollServiceMeshProxyDetailEndpoint(ctx)
				case <-initialize:
					for {
						c.UpdateClientHostEndpoints(ctx)
						err := c.PollServiceMeshProxyDetailEndpoint(ctx)
						if err == nil {
							break
						} else {
							time.Sleep(commons.ControlPlaneEndpointSleepInterval)
						}
					}
				}
			}
		}()
		return nil
	}))
}
