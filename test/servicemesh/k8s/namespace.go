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

func NewNamespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func CreateNamespace(ctx context.Context, client client.Client, key types.NamespacedName, ns *corev1.Namespace) error {
	err := client.Get(ctx, key, ns)
	if err != nil {
		err = client.Create(ctx, ns)
	}
	return err
}

func UpdateNamespace(ctx context.Context, client client.Client, ns *corev1.Namespace) error {
	return client.Update(ctx, ns)
}

func DeleteNamespace(ctx context.Context, client client.Client, ns *corev1.Namespace) error {
	return client.Delete(ctx, ns)
}

func UpdateProxyInjectionNamespaceLabel(ctx context.Context, client client.Client, ns *corev1.Namespace, value string) error {
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[ProxyInjectionLabel] = value
	return UpdateNamespace(ctx, client, ns)
}
