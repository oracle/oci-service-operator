/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceupdate

import (
	"context"
	"strconv"
	"time"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	podUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/pod"
	serviceUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/service"
	vdbUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/virtualdeploymentbinding"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceHandler interface {
	Reconcile(ctx context.Context, service *corev1.Service, namespaceInjectionLabel string) error
	FetchNamespaceInjectionLabel(ctx context.Context, req ctrl.Request, namespaceInjectionLabel *string) (bool, error)
	FetchService(ctx context.Context, namespacedName types.NamespacedName) (*corev1.Service, error)
}

type defaultResourceHandler struct {
	k8sClient client.Client
	log       loggerutil.OSOKLogger
	clientSet kubernetes.Interface
}

func NewDefaultResourceHandler(
	k8sClient client.Client,
	log loggerutil.OSOKLogger,
	clientSet kubernetes.Interface) ResourceHandler {
	return &defaultResourceHandler{
		k8sClient: k8sClient,
		log:       log,
		clientSet: clientSet,
	}
}

func (h *defaultResourceHandler) Reconcile(ctx context.Context, service *corev1.Service, namespaceInjectionLabel string) error {
	vdbs, err := vdbUtil.ListVDB(ctx, h.k8sClient)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return merrors.NewRequeueOnError(err)
	}

	filteredVDBs := vdbUtil.FilterVDBsByServiceRef(vdbs, service.Name, service.Namespace)
	if len(filteredVDBs) == 0 {
		h.log.InfoLogWithFixedMessage(ctx, "No VDB found with serviceRef")
		return nil
	}

	pods, err := serviceUtil.GetPodsForService(ctx, h.k8sClient, service)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return merrors.NewRequeueOnError(err)
	}
	h.log.InfoLogWithFixedMessage(ctx, "Fetched pods for the service", "number of pods fetched", strconv.Itoa(len(pods.Items)))

	return h.evictPods(ctx, pods, filteredVDBs, namespaceInjectionLabel)
}

func (h *defaultResourceHandler) FetchNamespaceInjectionLabel(ctx context.Context, req ctrl.Request, namespaceInjectionLabel *string) (bool, error) {
	namespace := &corev1.Namespace{}
	if err := h.k8sClient.Get(ctx, types.NamespacedName{Name: req.Namespace}, namespace); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, merrors.NewRequeueOnError(err)
	}

	namespaceLabels := labels.Set(namespace.Labels)
	if !namespaceLabels.Has(commons.ProxyInjectionLabel) {
		return false, nil
	}
	*namespaceInjectionLabel = namespaceLabels.Get(commons.ProxyInjectionLabel)
	return true, nil
}

func (h *defaultResourceHandler) FetchService(ctx context.Context, namespacedName types.NamespacedName) (*corev1.Service, error) {
	service := &corev1.Service{}
	if err := h.k8sClient.Get(ctx, namespacedName, service); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, merrors.NewDoNotRequeueError(err)
		}
		return nil, merrors.NewRequeueOnError(err)
	}

	if !service.DeletionTimestamp.IsZero() {
		return nil, merrors.NewDoNotRequeueError(errors.New("service is deleted"))
	}
	return service, nil
}

func (h *defaultResourceHandler) evictPods(ctx context.Context, pods *corev1.PodList, filteredVDBs []servicemeshapi.VirtualDeploymentBinding, namespaceInjectionLabel string) error {
	notEvictedPods := 0
	for i := range pods.Items {
		pod := &pods.Items[i]
		if podUtil.IsPodContainingServiceMeshProxy(pod) {
			continue
		}
		podLabels := labels.Set(pod.Labels)
		podInjectionLabel := ""
		if podLabels.Has(commons.ProxyInjectionLabel) {
			podInjectionLabel = podLabels.Get(commons.ProxyInjectionLabel)
		}

		if podUtil.IsInjectionLabelEnabled(namespaceInjectionLabel, podInjectionLabel) {
			matchedVDB := vdbUtil.GetVDBForPod(filteredVDBs, podLabels)
			if matchedVDB != nil {
				if err := podUtil.EvictPod(ctx, h.clientSet, pod); err != nil {
					h.log.ErrorLogWithFixedMessage(ctx, err, "Error in eviction", "pod", pod.Name)
					notEvictedPods += 1
				} else {
					h.log.InfoLogWithFixedMessage(ctx, "Pod eviction successful", "name", pod.Name)
				}
			}
		}
	}
	if notEvictedPods > 0 {
		h.log.InfoLogWithFixedMessage(ctx, "Pods are yet to be evicted, Reconciling after a minute", "count", strconv.Itoa(notEvictedPods))
		return merrors.NewRequeueAfter(time.Minute)
	}
	return nil
}
