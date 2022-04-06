/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/equality"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

func TestGetServiceDetails(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	var productService = NewKubernetesService("product", "test")
	err := CreateKubernetesService(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, productService)
	if err != nil {
		t.Fatal("Failure to create product service")
	}
	type args struct {
		serviceRef servicemeshapi.ResourceRef
	}
	type expected struct {
		service *corev1.Service
		err     error
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "GetKubernetesService",
			args: args{
				serviceRef: servicemeshapi.ResourceRef{
					Name:      "product",
					Namespace: "test",
				},
			},
			want: expected{
				service: productService,
				err:     nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GetKubernetesService(ctx, testFramework.K8sClient, tt.args.serviceRef, productService)
			if err != nil {
				assert.EqualError(t, tt.want.err, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPodsForService(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	var productService = NewKubernetesService("product", "test")
	err := CreateKubernetesService(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, productService)
	if err != nil {
		t.Fatal(fmt.Sprintf("Failure to create product service %v", err))
	}
	var productPod = NewPodWithLabels("product", "test", map[string]string{"app": "product"})
	err = CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, productPod)
	if err != nil {
		t.Fatal(fmt.Sprintf("Failure to create product pod %v", err))
	}

	type args struct {
		service *corev1.Service
	}
	type expected struct {
		pods *corev1.PodList
		err  error
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Get pod successful",
			args: args{
				service: productService,
			},
			want: expected{
				pods: &corev1.PodList{
					Items: []corev1.Pod{
						*productPod,
					},
				},
				err: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods, err := GetPodsForService(ctx, testFramework.K8sClient, tt.args.service)
			if err != nil {
				assert.EqualError(t, tt.want.err, err.Error())
			} else {
				opts := equality.IgnoreFakeClientPopulatedFields()
				assert.True(t, cmp.Equal(tt.want.pods, pods, opts), "diff", cmp.Diff(tt.want.pods, pods, opts))
			}
		})
	}
}
