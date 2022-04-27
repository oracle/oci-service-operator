/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package injectproxy

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	podUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"
	vdbUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/virtualdeploymentbinding"
)

type ResourceHandler interface {
	Reconcile(ctx context.Context, namespace *corev1.Namespace) error
}

type defaultResourceHandler struct {
	client             client.Client
	log                loggerutil.OSOKLogger
	clientSet          kubernetes.Interface
	configMapNamespace string
}

func NewDefaultResourceHandler(
	client client.Client,
	log loggerutil.OSOKLogger,
	clientSet kubernetes.Interface,
	configMapNamespace string) ResourceHandler {
	return &defaultResourceHandler{
		client:             client,
		log:                log,
		clientSet:          clientSet,
		configMapNamespace: configMapNamespace,
	}
}

// namespaceLabel   podLabel  			PodEligibilityForEviction
// enabled			enabled/""			true
// disabled			enabled				true
// enabled			disabled			false
// disabled			disabled/"" 		false
// ""				Not applicable		false
// If pod is eligible for proxy injection, it tries to match a VDB and if it finds a matching VDB it evicts the pod
// Do not requeue:
//	* If namespace does not contain proxy injection label
//	* If namespace deleting|deleted
//	* Contains empty pods,
//  * Contains empty vdb
//  * All pods in namespace is successfully evicted
// Requeue on Error:
//  * When Get|List operation fails
// Requeue after a minute:
//  * If pods could not be evicted due to PDBs.
//  A service with 100 pods with PDB 20 min available, it evicts 80 pods during first reconcile and it tries every minute for the remaining 20 pods to be evicted
func (h *defaultResourceHandler) Reconcile(ctx context.Context, namespace *corev1.Namespace) error {
	// Make sure sidecar image is populated in the config map before evicting the pod.
	configMap := &corev1.ConfigMap{}
	err := h.client.Get(ctx, types.NamespacedName{Namespace: h.configMapNamespace, Name: commons.MeshConfigMapName}, configMap)
	if err != nil {
		h.log.ErrorLog(err, "Error while fetching the configmap")
		return merrors.NewRequeueOnError(err)
	}
	sidecarImage, sidecarOk := configMap.Data[commons.ProxyLabelInMeshConfigMap]
	if !sidecarOk || len(sidecarImage) == 0 {
		err := errors.New("No sidecar image found in config map")
		h.log.ErrorLog(err, "No sidecar image found in config map")
		return merrors.NewRequeueOnError(err)
	}

	namespaceName := namespace.Name
	namespaceLabels := labels.Set(namespace.Labels)
	if !namespaceLabels.Has(commons.ProxyInjectionLabel) {
		h.log.InfoLog("Namespace does not have required labels", "label", commons.ProxyInjectionLabel)
		return merrors.NewRequeueAfter(time.Minute)
	}
	namespaceInjectionLabel := namespaceLabels.Get(commons.ProxyInjectionLabel)
	pods, err := podUtil.ListPods(ctx, h.client, namespaceName)
	if err != nil {
		if kerrors.IsNotFound(err) {
			h.log.InfoLog("Pods not found", "namespace", namespaceName)
			return nil
		}
		h.log.ErrorLog(err, "Error in listing the pods")
		return merrors.NewRequeueOnError(err)
	}

	virtualDeploymentBindingList, err := vdbUtil.ListVDB(ctx, h.client)
	if err != nil {
		if kerrors.IsNotFound(err) {
			h.log.InfoLog("VDBs not present")
			return nil
		}
		h.log.ErrorLog(err, "Error in listing virtualDeploymentBinding")
		return merrors.NewRequeueOnError(err)
	}

	notEvictedPods := 0
	for i := range pods.Items {
		if podUtil.IsPodContainingServiceMeshProxy(&pods.Items[i]) {
			continue
		}
		podLabels := labels.Set(pods.Items[i].Labels)
		podInjectionLabel := ""
		if podLabels.Has(commons.ProxyInjectionLabel) {
			podInjectionLabel = podLabels.Get(commons.ProxyInjectionLabel)
		}
		if podUtil.IsInjectionLabelEnabled(namespaceInjectionLabel, podInjectionLabel) {
			matchedVDB := podUtil.GetVDBForPod(ctx, h.client, &pods.Items[i], virtualDeploymentBindingList)
			if matchedVDB != nil {
				// TODO: Move it to a separate go routine to improve perf
				if err := podUtil.EvictPod(ctx, h.clientSet, &pods.Items[i]); err != nil {
					h.log.ErrorLog(err, "Error in eviction", "pod", pods.Items[i].Name)
					notEvictedPods += 1
				} else {
					h.log.InfoLog("Pod eviction successful", "name", pods.Items[i].Name)
				}
			}
		}
	}
	if notEvictedPods > 0 {
		h.log.InfoLog("Pods are yet to be evicted, Reconciling after a minute", "count", strconv.Itoa(notEvictedPods))
		return merrors.NewRequeueAfter(time.Minute)
	}

	return nil
}
