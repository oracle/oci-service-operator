/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package service

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

// Gets the kubernetes service object
func GetKubernetesService(ctx context.Context, r client.Reader, serviceRef servicemeshapi.ResourceRef, service *corev1.Service) error {
	return r.Get(ctx, types.NamespacedName{Namespace: serviceRef.Namespace, Name: string(serviceRef.Name)}, service)
}

// Gets the Pods for a given service
func GetPodsForService(ctx context.Context, cl client.Client, service *corev1.Service) (*corev1.PodList, error) {
	serviceLabels := labels.Set(service.Spec.Selector)
	pods := &corev1.PodList{}
	listOptions := client.ListOptions{LabelSelector: serviceLabels.AsSelector()}
	err := cl.List(ctx, pods, &listOptions)
	return pods, err
}

// IsServiceActive tests whether given service is active.
func IsServiceActive(service *corev1.Service) bool {
	// TODO: check the service status object upon migration to kube v1.20
	return true
}

// GetServiceActiveStatus returns the current status of the VD
func GetServiceActiveStatus(service *corev1.Service) metav1.ConditionStatus {
	// TODO: check the service status upon migration to kube v1.20
	return metav1.ConditionUnknown
}
