/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	namespaceUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
)

func NewEnqueueRequestsForConfigmapEvents(k8sClient client.Client, log logr.Logger, namespace string) *enqueueRequestsForConfigmapEvents {
	return &enqueueRequestsForConfigmapEvents{
		k8sClient: k8sClient,
		log:       log,
		namespace: namespace,
	}
}

var _ handler.EventHandler = (*enqueueRequestsForConfigmapEvents)(nil)

type enqueueRequestsForConfigmapEvents struct {
	k8sClient client.Client
	log       logr.Logger
	namespace string
}

func (e enqueueRequestsForConfigmapEvents) Create(event event.CreateEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (e enqueueRequestsForConfigmapEvents) Update(event event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	e.log.Info("New ConfigMap Update Received")
	cNew := event.ObjectNew.(*corev1.ConfigMap)
	if cNew.Name == commons.MeshConfigMapName && cNew.Namespace == e.namespace {
		e.enqueueIngressGatewayDeploymentsForConfigMap(context.Background(), queue, cNew)
	}
}

func (e enqueueRequestsForConfigmapEvents) Delete(event event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (e enqueueRequestsForConfigmapEvents) Generic(event event.GenericEvent, queue workqueue.RateLimitingInterface) {
	// no-op
}

func (e *enqueueRequestsForConfigmapEvents) enqueueIngressGatewayDeploymentsForConfigMap(ctx context.Context, queue workqueue.RateLimitingInterface, configMap *corev1.ConfigMap) {
	e.log.Info("ConfigMap Update Received")
	ingressList := &servicemeshapi.IngressGatewayDeploymentList{}
	if err := e.k8sClient.List(ctx, ingressList); err != nil {
		e.log.Error(err, "failed to enqueue Ingress Deployments for configMap events",
			"configMap", namespaceUtil.NewNamespacedName(configMap))
		return
	}
	for i := range ingressList.Items {
		e.log.Info("In Queue " + ingressList.Items[i].Name)
		queue.Add(ctrl.Request{NamespacedName: namespaceUtil.NewNamespacedName(&ingressList.Items[i])})
	}
}
