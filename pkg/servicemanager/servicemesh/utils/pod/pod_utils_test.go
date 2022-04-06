/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package pod

import (
	"context"
	"testing"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

var productPodWithServiceMeshProxy = *NewPodWithServiceMeshProxy("product", "test")

func TestIsPodContainingServiceMeshProxy(t *testing.T) {
	type args struct {
		pod corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Pod contains service mesh proxy",
			args: args{
				pod: productPodWithServiceMeshProxy,
			},
			want: true,
		},
		{
			name: "Pod does not contains service mesh proxy",
			args: args{
				pod: *NewPodWithoutServiceMeshProxy("product", "test"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := IsPodContainingServiceMeshProxy(&tt.args.pod)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestGetCurrentProxyVersion(t *testing.T) {
	type args struct {
		pod corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Get pod proxy image version when it contains service mesh proxy",
			args: args{
				pod: productPodWithServiceMeshProxy,
			},
			want: "sm-proxy-image",
		},
		{
			name: "Get pod proxy image version when it not contains service mesh proxy",
			args: args{
				pod: *NewPodWithoutServiceMeshProxy("product", "test"),
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := GetProxyVersion(&tt.args.pod)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestIsInjectionLabelEnabled(t *testing.T) {
	type args struct {
		namespaceLabel string
		podLabel       string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ProxyInjectionLabel Namespace:Disabled Pod:Empty",
			args: args{
				namespaceLabel: Disabled,
				podLabel:       "",
			},
			want: false,
		},
		{
			name: "ProxyInjectionLabel Namespace:Disabled Pod:Disabled",
			args: args{
				namespaceLabel: Disabled,
				podLabel:       Disabled,
			},
			want: false,
		},
		{
			name: "ProxyInjectionLabel Namespace:Disabled Pod:Enabled",
			args: args{
				namespaceLabel: Disabled,
				podLabel:       Enabled,
			},
			want: true,
		},
		{
			name: "ProxyInjectionLabel Namespace:Enabled Pod:Empty",
			args: args{
				namespaceLabel: Enabled,
				podLabel:       "",
			},
			want: true,
		},
		{
			name: "ProxyInjectionLabel Namespace:Enabled Pod:Enabled",
			args: args{
				namespaceLabel: Enabled,
				podLabel:       Enabled,
			},
			want: true,
		},
		{
			name: "ProxyInjectionLabel Namespace:Enabled Pod:Invalid",
			args: args{
				namespaceLabel: Enabled,
				podLabel:       "Invalid",
			},
			want: false,
		},
		{
			name: "ProxyInjectionLabel Namespace:Disabled Pod:Invalid",
			args: args{
				namespaceLabel: Disabled,
				podLabel:       "Invalid",
			},
			want: false,
		},
		{
			name: "ProxyInjectionLabel Namespace:random Pod:Invalid",
			args: args{
				namespaceLabel: "random",
				podLabel:       "Invalid",
			},
			want: false,
		},
		{
			name: "ProxyInjectionLabel Namespace:Enabled Pod:Disabled",
			args: args{
				namespaceLabel: Enabled,
				podLabel:       Disabled,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := IsInjectionLabelEnabled(tt.args.namespaceLabel, tt.args.podLabel)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestGetVDBForPod(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	var productService = NewKubernetesService("product", "test")
	err := CreateKubernetesService(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, productService)
	if err != nil {
		t.Fatal("Failure to create product service")
	}
	type args struct {
		virtualDeploymentBindingList *servicemeshapi.VirtualDeploymentBindingList
	}
	type expected struct {
		value *servicemeshapi.VirtualDeploymentBinding
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Empty Virtual Deployment Binding",
			args: args{
				virtualDeploymentBindingList: &servicemeshapi.VirtualDeploymentBindingList{Items: make([]servicemeshapi.VirtualDeploymentBinding, 0)},
			},
			want: expected{
				value: nil,
			},
		},
		{
			name: "PodUpgradeEnabled:true and Pod matches Virtual Deployment Binding",
			args: args{
				virtualDeploymentBindingList: getVirtualDeploymentBindingList(),
			},
			want: expected{
				value: NewVdbWithVdRef("product-v1-binding", "test", "product", "product"),
			},
		},
		{
			name: "PodUpgradeEnabled:false and Pod matches Virtual Deployment Binding",
			args: args{
				virtualDeploymentBindingList: getVirtualDeploymentBindingList(),
			},
			want: expected{
				value: NewVdbWithVdRef("product-v1-binding", "test", "product", "product"),
			},
		},
		{
			name: "Pod matches multiple Virtual Deployment Binding",
			args: args{
				virtualDeploymentBindingList: getMultipleVirtualDeploymentBindingMatch(),
			},
			want: expected{
				value: NewVdbWithVdRef("product-v1-binding", "test", "product", "product"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := GetVDBForPod(ctx, testFramework.K8sClient, &productPodWithServiceMeshProxy, tt.args.virtualDeploymentBindingList)
			if value == nil {
				assert.Nil(t, tt.want.value)
			} else {
				assert.Equal(t, tt.want.value.Name, value.Name)
			}
		})
	}
}

func TestListPods(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	err := CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, &productPodWithServiceMeshProxy)
	if err != nil {
		t.Fatal("Failure to create product pod")
	}
	podItems := make([]corev1.Pod, 0)
	podItems = append(podItems, productPodWithServiceMeshProxy)
	podList := corev1.PodList{
		Items: podItems,
	}
	type args struct {
		Namespace string
	}
	type expected struct {
		value corev1.PodList
		err   error
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "List pods",
			args: args{
				Namespace: "test",
			},
			want: expected{
				value: podList,
				err:   nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podList, err := ListPods(ctx, testFramework.K8sClient, tt.args.Namespace)
			assert.NoError(t, err)
			assert.Len(t, podList.Items, 1)
			assert.True(t, cmp.Equal(tt.want.value.Items, podList.Items, nil), "diff", cmp.Diff(tt.want.value.Items, podList.Items, nil))
		})
	}
}

func TestIsPodBelongsToVirtualDeploymentBinding(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	var productService = NewKubernetesService("product", "test")
	selectorLabels := productService.Spec.Selector
	selectorLabels["version"] = "v2"
	err := CreateKubernetesService(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, productService)
	if err != nil {
		t.Fatal("Failure to create product service")
	}
	type args struct {
		virtualDeploymentBinding servicemeshapi.VirtualDeploymentBinding
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Virtual Deployment Binding Match labels did not match pod labels",
			args: args{
				virtualDeploymentBinding: *getVirtualDeploymentBindingWherePodLabelsNotMatchVDBMatchingLabels(),
			},
			want: false,
		},
		{
			name: "Pod labels did not match selector labels",
			args: args{
				virtualDeploymentBinding: *NewVdbWithVdRef("product-v1-binding", "test", "product", "product"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := IsPodBelongsToVDB(ctx, testFramework.K8sClient, &productPodWithServiceMeshProxy, &tt.args.virtualDeploymentBinding)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestSetOutdatedProxyAnnotation(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	productPodWithServiceMeshProxy = *NewPodWithServiceMeshProxy("product", "test")
	err := CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, &productPodWithServiceMeshProxy)
	if err != nil {
		t.Fatal("Failure to create product pod")
	}
	var podWithAnnotations = *NewPodWithServiceMeshProxy("product", "test")
	podWithAnnotations.Annotations = map[string]string{
		OutdatedProxyAnnotation: "true",
	}
	err = CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, &podWithAnnotations)
	if err != nil {
		t.Fatal("Failure to create product pod")
	}
	var podWithExistingAnnotations = *NewPodWithServiceMeshProxy("product", "test")
	podWithExistingAnnotations.Annotations = map[string]string{
		OutdatedProxyAnnotation: "true",
	}
	err = CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product"}, &podWithExistingAnnotations)
	if err != nil {
		t.Fatal("Failure to create pod with annotations")
	}
	tests := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{
			name: "Pod contains empty annotations",
			pod:  &productPodWithServiceMeshProxy,
			want: true,
		},
		{
			name: "Pod already contains outdated proxy annotations",
			pod:  &podWithAnnotations,
			want: true,
		},
		{
			name: "Pod contains annotations",
			pod:  &podWithExistingAnnotations,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := SetOutdatedProxyAnnotation(ctx, testFramework.K8sClient, tt.pod)
			assert.Equal(t, tt.want, value)
			err := GetPod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: tt.pod.Name}, tt.pod)
			if err != nil {
				t.Fatal(err, "Error in fetching pod")
			}
			assert.Equal(t, tt.pod.Annotations[OutdatedProxyAnnotation], "true")
		})
	}
}

func getVirtualDeploymentBindingList() *servicemeshapi.VirtualDeploymentBindingList {
	virtualDeploymentBindingList := make([]servicemeshapi.VirtualDeploymentBinding, 0)
	virtualDeploymentBindingList = append(virtualDeploymentBindingList, *NewVdbWithVdRef("product-v1-binding", "test", "product", "product"))
	return &servicemeshapi.VirtualDeploymentBindingList{Items: virtualDeploymentBindingList}
}

func getMultipleVirtualDeploymentBindingMatch() *servicemeshapi.VirtualDeploymentBindingList {
	virtualDeploymentBindingList := getVirtualDeploymentBindingList()
	virtualDeploymentBindingList.Items = append(virtualDeploymentBindingList.Items, *NewVdbWithVdRef("product-v2-binding", "test", "product-v2", "product"))
	return virtualDeploymentBindingList
}

func getVirtualDeploymentBindingWherePodLabelsNotMatchVDBMatchingLabels() *servicemeshapi.VirtualDeploymentBinding {
	virtualDeploymentBinding := NewVdbWithVdRef("product-v1-binding", "test", "product", "product")
	virtualDeploymentBinding.Spec.Target.Service.MatchLabels = map[string]string{
		"matchLabel": "notMatch",
	}
	return virtualDeploymentBinding
}
