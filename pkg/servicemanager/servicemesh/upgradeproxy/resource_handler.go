/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package upgradeproxy

import (
	"context"
	"strconv"
	"time"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	namespaceUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/namespace"
	podUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"
	vdbUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/virtualdeploymentbinding"
)

type ResourceHandler interface {
	Reconcile(ctx context.Context, configMap *corev1.ConfigMap) error
}

type defaultResourceHandler struct {
	client    client.Client
	log       loggerutil.OSOKLogger
	clientSet kubernetes.Interface
}

func NewDefaultResourceHandler(
	client client.Client,
	log loggerutil.OSOKLogger,
	clientSet kubernetes.Interface) ResourceHandler {
	return &defaultResourceHandler{
		client:    client,
		log:       log,
		clientSet: clientSet,
	}
}

// 1. Lists all service mesh enabled namespaces
// 2. For each namespace get all its corresponding pods
// 3. For each pod check if the pod has service mesh proxy, if it has one check for version mismatch from configmap value, if mismatch perform following steps
//   3.1 Get vdb-ref from the pod annotation
//   3.3 Else add outdated proxy annotation to the pod
func (h *defaultResourceHandler) Reconcile(ctx context.Context, configMap *corev1.ConfigMap) error {
	newProxyVersion, newProxyVersionPresent := configMap.Data[commons.ProxyLabelInMeshConfigMap]
	h.log.InfoLog("Proxy version have been updated", "New Proxy Version", newProxyVersion)
	if !newProxyVersionPresent {
		h.log.InfoLog("No sidecar image found in mesh config map")
		return nil
	}
	serviceMeshEnabledNamespaces, err := namespaceUtil.ListServiceMeshEnabledNamespaces(ctx, h.client)
	if err != nil {
		if errors.IsNotFound(err) {
			h.log.InfoLog("No mesh enabled namespace found")
			return nil
		}
		h.log.ErrorLog(err, "Error in listing mesh enabled namespaces")
		return merrors.NewRequeueOnError(err)
	}
	notEvictedNamespaces := 0
	notEvictedPods := 0
	for _, namespace := range serviceMeshEnabledNamespaces.Items {
		podList, err := podUtil.ListPods(ctx, h.client, namespace.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				h.log.InfoLog("Pods not found", "namespace", namespace.Name)
				continue
			}
			h.log.ErrorLog(err, "Error in listing the pods", "namespace", namespace.Name)
			notEvictedNamespaces += 1
			continue
		}
		for i := range podList.Items {
			currentProxyVersion := podUtil.GetProxyVersion(&podList.Items[i])
			if currentProxyVersion == "" || currentProxyVersion == newProxyVersion {
				continue
			}
			vdbRef := podList.Items[i].Annotations[commons.VirtualDeploymentBindingAnnotation]
			vdbRefNsName, valid := namespaceUtil.NewNamespacedNameFromString(vdbRef)
			if !valid {
				if !podUtil.SetOutdatedProxyAnnotation(ctx, h.client, &podList.Items[i]) {
					notEvictedPods += 1
				}
				continue
			}
			vdb := servicemeshapi.VirtualDeploymentBinding{}
			err = vdbUtil.GetVDB(ctx, h.client, vdbRefNsName.Namespace, vdbRefNsName.Name, &vdb)
			if err != nil {
				if errors.IsNotFound(err) {
					h.log.InfoLog("VDB not found", "namespace", vdbRefNsName.Namespace, "name", vdbRefNsName.Name)
					continue
				}
				notEvictedPods += 1
				continue
			}

			// TODO: Move it to a separate go routine to improve perf
			if err := podUtil.EvictPod(ctx, h.clientSet, &podList.Items[i]); err != nil {
				h.log.ErrorLog(err, "Error in eviction", "pod", podList.Items[i].Name, "Namespace", podList.Items[i].Namespace)
				notEvictedPods += 1
			} else {
				h.log.InfoLog("Pod evicted successfully", "name", podList.Items[i].Name, "Namespace", podList.Items[i].Namespace)
			}
		}
	}
	if notEvictedNamespaces > 0 || notEvictedPods > 0 {
		h.log.InfoLog("Pods are yet to be evicted, Reconciling after a minute", "NamespaceCount", notEvictedNamespaces, "PodsCount", strconv.Itoa(notEvictedPods))
		return merrors.NewRequeueAfter(time.Minute)
	}
	h.log.InfoLog("All pods are upgraded successfully to latest proxy version")
	return nil
}
