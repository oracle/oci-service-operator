/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pod

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	serviceUtil "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/service"
)

var logger = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("PodUtils")}

// Returns true if pod contains service mesh proxy
// Returns false if pod does not contain service mesh proxy
func IsPodContainingServiceMeshProxy(pod *corev1.Pod) (valid bool) {
	for _, containers := range pod.Spec.Containers {
		if containers.Name == commons.ProxyContainerName {
			logger.InfoLog(fmt.Sprintf("Pod %s contains service mesh proxy", pod.Name))
			return true
		}
	}
	return
}

// Returns a boolean based on pod eligibility for proxy injection
// Input - labels - pod labels, namespaceProxyInjectionLabelValue - proxy injection label at namespace
// NamespaceLabel   PodLabel  			PodEligibility
// enabled			enabled/""			true
// disabled			enabled				true
// enabled			disabled			false
// disabled			disabled/"" 		false
// ""				Not applicable		false
func IsInjectionLabelEnabled(namespaceLabel string, podLabel string) bool {
	if namespaceLabel == commons.Enabled && (podLabel == commons.Enabled || podLabel == "") {
		return true
	}
	if namespaceLabel == commons.Disabled && podLabel == commons.Enabled {
		return true
	}
	return false
}

// Get proxy version for a given pod
func GetProxyVersion(pod *corev1.Pod) (value string) {
	for _, containers := range pod.Spec.Containers {
		if containers.Name == commons.ProxyContainerName {
			return containers.Image
		}
	}
	return
}

// Set outdated proxy annotation for a given pod if pod proxy version is not upgraded to the latest proxy version
func SetOutdatedProxyAnnotation(ctx context.Context, cl client.Client, pod *corev1.Pod) bool {
	newPod := pod.DeepCopy()
	annotations := newPod.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	} else {
		_, ok := annotations[commons.OutdatedProxyAnnotation]
		if ok {
			logger.InfoLog("contains outdated-proxy annotation", "Pod", pod.Name, "Namespace", pod.Namespace)
			return true
		}
	}
	annotations[commons.OutdatedProxyAnnotation] = "true"
	newPod.ObjectMeta.Annotations = annotations
	return updatePodAnnotation(ctx, cl, newPod, pod)
}

func updatePodAnnotation(ctx context.Context, cl client.Client, newPod *corev1.Pod, pod *corev1.Pod) (valid bool) {
	err := cl.Status().Patch(ctx, newPod, client.MergeFrom(pod))
	if err != nil {
		logger.ErrorLog(err, "Error in updating pod annotation", "Pod", pod.Name, "Namespace", pod.Namespace)
		return
	}
	logger.InfoLog("Successfully updated pod annotation", "Pod", pod.Name, "Namespace", pod.Namespace)
	return true
}

// Returns list of pods for a given namespace
func ListPods(ctx context.Context, cl client.Client, namespace string) (*corev1.PodList, error) {
	return ListPodsWithLabels(ctx, cl, namespace, &labels.Set{})
}

// Returns list of pods for given selectors and namespace
func ListPodsWithLabels(ctx context.Context, cl client.Client, namespace string, labels *labels.Set) (*corev1.PodList, error) {
	podsList := &corev1.PodList{}
	listOpts := &client.ListOptions{LabelSelector: labels.AsSelector()}
	listOpts.Namespace = namespace
	err := cl.List(ctx, podsList, listOpts)
	return podsList, err
}

// Evict a pod for a given namespace
func EvictPod(ctx context.Context, clientSet kubernetes.Interface, pod *corev1.Pod) error {
	return clientSet.CoreV1().Pods(pod.Namespace).Evict(ctx, &policyv1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
	})
}

// returns a matching VDB for a pod from vdbList
func GetVDBForPod(ctx context.Context, cl client.Reader, pod *corev1.Pod, vdbs *servicemeshapi.VirtualDeploymentBindingList) *servicemeshapi.VirtualDeploymentBinding {
	for i := range vdbs.Items {
		if IsPodBelongsToVDB(ctx, cl, pod, &vdbs.Items[i]) {
			// return the first matched VDB
			logger.InfoLog(fmt.Sprintf("Pod: %s in Namespace: %s matches VirtualDeploymentBinding", pod.Name, pod.Namespace))
			return &vdbs.Items[i]
		}
	}
	return nil
}

// checks if a pod belongs to a vdb
func IsPodBelongsToVDB(ctx context.Context, cl client.Reader, pod *corev1.Pod, virtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding) (valid bool) {
	serviceRef := virtualDeploymentBinding.Spec.Target.Service.ServiceRef
	if serviceRef.Namespace == "" {
		serviceRef.Namespace = virtualDeploymentBinding.Namespace
	}
	if serviceRef.Namespace != pod.Namespace {
		return
	}
	matchLabels := virtualDeploymentBinding.Spec.Target.Service.MatchLabels
	if !MatchLabels(pod.Labels, matchLabels) {
		return
	}

	service := corev1.Service{}
	if err := serviceUtil.GetKubernetesService(ctx, cl, serviceRef, &service); err != nil {
		logger.InfoLog(fmt.Sprintf("Error in fetching service for Name: %s Namespace: %s", serviceRef.Name, serviceRef.Namespace))
		return
	}
	serviceSelectorLabels := labels.Set(service.Spec.Selector)
	if !MatchLabels(pod.Labels, serviceSelectorLabels) {
		return
	}

	return true
}

// checks if all the labels are present in podLabels
func MatchLabels(podLabels labels.Set, labels labels.Set) (valid bool) {
	for k, v := range labels {
		if !podLabels.Has(k) || (podLabels.Get(k) != v) {
			return
		}
	}
	return true
}

func GetPodInjectionLabelValue(podLabels *labels.Set) string {
	podInjectionLabel := ""
	if podLabels.Has(commons.ProxyInjectionLabel) {
		podInjectionLabel = podLabels.Get(commons.ProxyInjectionLabel)
	}
	return podInjectionLabel
}
