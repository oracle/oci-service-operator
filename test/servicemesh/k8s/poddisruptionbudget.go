/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package k8s

import (
	"context"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPodDisruptionBudget(name string, namespace string, minAvailable int32, target string) *policyv1beta1.PodDisruptionBudget {
	return &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: minAvailable,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": target},
			},
		},
	}
}

func CreatePodDisruptionBudget(ctx context.Context, client client.Client, key types.NamespacedName, pdb *policyv1beta1.PodDisruptionBudget) error {
	err := client.Get(ctx, key, pdb)
	if err != nil {
		err = client.Create(ctx, pdb)
	}
	return err
}

func DeletePodDisruptionBudget(ctx context.Context, client client.Client, pdb *policyv1beta1.PodDisruptionBudget) error {
	return client.Delete(ctx, pdb)
}
