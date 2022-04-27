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
)

func NewConfigMap(namespace string, name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func CreateConfigMap(ctx context.Context, client client.Client, key types.NamespacedName, cm *corev1.ConfigMap) error {
	err := client.Get(ctx, key, cm)
	if err != nil {
		err = client.Create(ctx, cm)
	}
	return err
}

func UpdateConfigMap(ctx context.Context, client client.Client, cm *corev1.ConfigMap) error {
	return client.Update(ctx, cm)
}
