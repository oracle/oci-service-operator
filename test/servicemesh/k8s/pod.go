/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

func NewPodWithServiceMeshProxy(name string, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: name + "-image",
				},
				{
					Name:  ProxyContainerName,
					Image: "sm-proxy-image",
				},
			},
		},
	}
}

func NewPodWithoutServiceMeshProxy(name string, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: name + "-image",
				},
			},
		},
	}
}

func NewPodWithLabels(name string, namespace string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: name + "-image",
				},
			},
		},
	}
}

func AddPodConditionReady(pod *corev1.Pod) {
	pod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
}

func CreatePod(ctx context.Context, client client.Client, key types.NamespacedName, pod *corev1.Pod) error {
	err := client.Get(ctx, key, pod)
	if err != nil {
		err = client.Create(ctx, pod)
	}
	return err
}

func UpdatePod(ctx context.Context, client client.Client, pod *corev1.Pod) error {
	return client.Update(ctx, pod)
}

func GetPod(ctx context.Context, client client.Client, key types.NamespacedName, pod *corev1.Pod) error {
	return client.Get(ctx, key, pod)
}

func UpdateProxyInjectionPodLabel(ctx context.Context, client client.Client, pod *corev1.Pod, value string) error {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[ProxyInjectionLabel] = value
	return UpdatePod(ctx, client, pod)
}

func UpdateVDBRefPodAnnotation(ctx context.Context, client client.Client, pod *corev1.Pod, value string) error {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[VirtualDeploymentBindingAnnotation] = value
	return UpdatePod(ctx, client, pod)
}
