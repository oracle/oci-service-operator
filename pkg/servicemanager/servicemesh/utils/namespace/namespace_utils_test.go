/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package namespace

import (
	"context"
	"testing"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
)

func TestListNamespaces(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	testNamespace1 := *NewNamespace("test1", map[string]string{commons.ProxyInjectionLabel: commons.Enabled})
	err := CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testNamespace1.Name}, &testNamespace1)
	if err != nil {
		t.Fatal("Failure to create test1 namespace")
	}
	testNamespace2 := *NewNamespace("test2", map[string]string{commons.ProxyInjectionLabel: ""})
	err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testNamespace2.Name}, &testNamespace2)
	if err != nil {
		t.Fatal("Failure to create test2 namespace")
	}
	testNamespace3 := *NewNamespace("test3", map[string]string{commons.ProxyInjectionLabel: commons.Disabled})
	err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testNamespace3.Name}, &testNamespace3)
	if err != nil {
		t.Fatal("Failure to create test3 namespace")
	}
	testNamespace4 := *NewNamespace("test4", map[string]string{})
	err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testNamespace4.Name}, &testNamespace4)
	if err != nil {
		t.Fatal("Failure to create test4 namespace")
	}
	namespaceItems := make([]corev1.Namespace, 0)
	namespaceItems = append(namespaceItems, testNamespace1, testNamespace2, testNamespace3)
	namespaceList := corev1.NamespaceList{
		Items: namespaceItems,
	}
	type expected struct {
		namespaceList corev1.NamespaceList
		err           error
	}
	tests := []struct {
		name string
		want expected
	}{
		{
			name: "Get Service mesh enabled namespaces",
			want: expected{
				namespaceList: namespaceList,
				err:           nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceMeshEnabledNamespacesList, err := ListServiceMeshEnabledNamespaces(ctx, testFramework.K8sClient)
			assert.NoError(t, err)
			assert.Len(t, serviceMeshEnabledNamespacesList.Items, 3)
			assert.True(t, cmp.Equal(tt.want.namespaceList.Items, serviceMeshEnabledNamespacesList.Items, nil), "diff", cmp.Diff(tt.want.namespaceList.Items, serviceMeshEnabledNamespacesList.Items, nil))
		})
	}
}

func TestNewNamespacedNameFromString(t *testing.T) {
	type args struct {
		value string
	}
	type expected struct {
		value types.NamespacedName
		valid bool
	}
	tests := []struct {
		name string
		args args
		want expected
	}{
		{
			name: "Empty",
			args: args{
				value: "",
			},
			want: expected{
				value: types.NamespacedName{},
				valid: false,
			},
		},
		{
			name: "Valid",
			args: args{
				value: "ns/name",
			},
			want: expected{
				value: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
				valid: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, valid := NewNamespacedNameFromString(tt.args.value)
			assert.Equal(t, tt.want.valid, valid)
			assert.Equal(t, tt.want.value, value)
		})
	}
}

func TestNewNamespacedName(t *testing.T) {
	tests := []struct {
		name string
		mesh *servicemeshapi.Mesh
		want types.NamespacedName
	}{
		{
			name: "NamespacedName with name and namespace",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mesh",
					Namespace: "namespace",
				},
			},
			want: types.NamespacedName{
				Name:      "mesh",
				Namespace: "namespace",
			},
		},
		{
			name: "NamespacedName without namespace",
			mesh: &servicemeshapi.Mesh{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mesh",
				},
			},
			want: types.NamespacedName{
				Name: "mesh",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := NewNamespacedName(tt.mesh)
			assert.Equal(t, tt.want, value)
		})
	}
}
