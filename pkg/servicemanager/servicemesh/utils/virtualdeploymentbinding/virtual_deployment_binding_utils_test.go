/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	kubernetesScheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/oracle/oci-service-operator/test/servicemesh/virtualdeploymentbinding"
)

var vdbVersionV1 = servicemeshapi.VirtualDeploymentBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "test-ns-vdb",
		Name:      "test-vdb",
	},
	Spec: servicemeshapi.VirtualDeploymentBindingSpec{
		Target: servicemeshapi.Target{
			Service: servicemeshapi.Service{
				ServiceRef: servicemeshapi.ResourceRef{
					Name: "test-service",
				},
				MatchLabels: map[string]string{
					"version": "v1",
				},
			},
		},
	},
}

var vdbVersionV2 = servicemeshapi.VirtualDeploymentBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "test-ns-vdb",
		Name:      "test-vdb-2",
	},
	Spec: servicemeshapi.VirtualDeploymentBindingSpec{
		Target: servicemeshapi.Target{
			Service: servicemeshapi.Service{
				ServiceRef: servicemeshapi.ResourceRef{
					Name: "test-service-2",
				},
				MatchLabels: map[string]string{
					"version": "v2",
				},
			},
		},
	},
}
var vdbVersionV3 = servicemeshapi.VirtualDeploymentBinding{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "test-ns-vdb",
		Name:      "test-vdb-3",
	},
	Spec: servicemeshapi.VirtualDeploymentBindingSpec{
		Target: servicemeshapi.Target{
			Service: servicemeshapi.Service{
				ServiceRef: servicemeshapi.ResourceRef{
					Name: "test-service-2",
				},
				MatchLabels: map[string]string{
					"version": "v1",
					"app":     "product",
				},
			},
		},
	},
}

func TestListVDB(t *testing.T) {
	ctx := context.Background()
	testFramework := framework.NewFakeClientFramework(t)
	productVirtualDeploymentBinding := NewVdbWithVdRef("product-v1-binding", "test", "product", "product")
	err := CreateVirtualDeploymentBinding(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: "test", Name: "product-v1-binding"}, productVirtualDeploymentBinding)
	if err != nil {
		t.Fatal("Failure to create virtual deployment binding")
	}
	virtualDeploymentBindingItems := make([]servicemeshapi.VirtualDeploymentBinding, 0)
	virtualDeploymentBindingItems = append(virtualDeploymentBindingItems, *productVirtualDeploymentBinding)
	virtualDeploymentBindingList := servicemeshapi.VirtualDeploymentBindingList{
		Items: virtualDeploymentBindingItems,
	}
	type expected struct {
		virtualDeploymentBindingList servicemeshapi.VirtualDeploymentBindingList
		err                          error
	}
	tests := []struct {
		name string
		want expected
	}{
		{
			name: "ListVirtualDeploymentBindingList",
			want: expected{
				virtualDeploymentBindingList: virtualDeploymentBindingList,
				err:                          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			virtualDeploymentBindingList, err := ListVDB(ctx, testFramework.K8sClient)
			assert.NoError(t, err)
			assert.Len(t, virtualDeploymentBindingList.Items, 1)
			assert.True(t, cmp.Equal(tt.want.virtualDeploymentBindingList.Items, virtualDeploymentBindingList.Items, nil), "diff", cmp.Diff(tt.want.virtualDeploymentBindingList.Items, virtualDeploymentBindingList.Items, nil))
		})
	}
}

func TestGetVDB(t *testing.T) {
	ctx := context.Background()
	scheme := kubernetesScheme.Scheme
	_ = servicemeshapi.AddToScheme(scheme)
	client := testclient.NewClientBuilder().WithScheme(scheme).Build()
	productVDB := NewVdbWithVdRef("product-v1-binding", "test", "product", "product")
	err := CreateVirtualDeploymentBinding(ctx, client, types.NamespacedName{Namespace: "test", Name: "product-v1-binding"}, productVDB)
	if err != nil {
		t.Fatal(fmt.Sprintf("Failure to create virtual deployment binding %v", err))
	}
	tests := []struct {
		name string
		want *servicemeshapi.VirtualDeploymentBinding
	}{
		{
			name: "GetVirtualDeploymentBinding",
			want: productVDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdb := servicemeshapi.VirtualDeploymentBinding{}
			err := GetVDB(ctx, client, productVDB.Namespace, productVDB.Name, &vdb)
			assert.NoError(t, err)
		})
	}
}

func TestGetServiceRefNamespace(t *testing.T) {
	type args struct {
		vdb *servicemeshapi.VirtualDeploymentBinding
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "service ref namespace present",
			args: args{
				vdb: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns-vdb",
						Name:      "test-vdb",
					},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						Target: servicemeshapi.Target{
							Service: servicemeshapi.Service{
								ServiceRef: servicemeshapi.ResourceRef{
									Name:      "test-service",
									Namespace: "test",
								},
							},
						},
					},
				},
			},
			want: "test",
		},
		{
			name: "service ref namespace absent",
			args: args{
				vdb: &servicemeshapi.VirtualDeploymentBinding{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns-vdb",
						Name:      "test-vdb",
					},
					Spec: servicemeshapi.VirtualDeploymentBindingSpec{
						Target: servicemeshapi.Target{
							Service: servicemeshapi.Service{
								ServiceRef: servicemeshapi.ResourceRef{
									Name: "test-service",
								},
							},
						},
					},
				},
			},
			want: "test-ns-vdb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetServiceRefNamespace(tt.args.vdb); got != tt.want {
				t.Errorf("GetServiceRefNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVirtualDeploymentBindingActive(t *testing.T) {
	type args struct {
		virtualDeploymentBinding *servicemeshapi.VirtualDeploymentBinding
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "vdbActive true",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshDependenciesActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "vdbActive false",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{
								Type: servicemeshapi.ServiceMeshActive,
								ResourceCondition: servicemeshapi.ResourceCondition{
									Status: metav1.ConditionFalse,
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "vdbActive not present",
			args: args{
				virtualDeploymentBinding: &servicemeshapi.VirtualDeploymentBinding{
					Status: servicemeshapi.ServiceMeshStatus{
						Conditions: []servicemeshapi.ServiceMeshCondition{
							{},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVirtualDeploymentBindingActive(tt.args.virtualDeploymentBinding); got != tt.want {
				t.Errorf("IsVirtualDeploymentBindingActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVDBForPod(t *testing.T) {
	type args struct {
		vdbs      []servicemeshapi.VirtualDeploymentBinding
		podLabels labels.Set
	}
	tests := []struct {
		name string
		args args
		want *servicemeshapi.VirtualDeploymentBinding
	}{
		{
			name: "single vdb found for a pod",
			args: args{
				vdbs: []servicemeshapi.VirtualDeploymentBinding{
					vdbVersionV1,
					vdbVersionV2,
					vdbVersionV3,
				},
				podLabels: labels.Set{
					"version": "v1",
				},
			},
			want: &vdbVersionV1,
		},
		{
			name: "no vdb found for a pod",
			args: args{
				vdbs: []servicemeshapi.VirtualDeploymentBinding{
					vdbVersionV1,
					vdbVersionV2,
					vdbVersionV3,
				},
				podLabels: labels.Set{
					"version": "v5",
					"app":     "product",
				},
			},
			want: nil,
		},
		{
			name: "single vdb found for a pod",
			args: args{
				vdbs: []servicemeshapi.VirtualDeploymentBinding{
					vdbVersionV1,
					vdbVersionV2,
					vdbVersionV3,
				},
				podLabels: labels.Set{
					"version": "v1",
					"app":     "product",
				},
			},
			want: &vdbVersionV1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVDBForPod(tt.args.vdbs, tt.args.podLabels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVDBForPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterVDBsByServiceRef(t *testing.T) {
	type args struct {
		vdbs             *servicemeshapi.VirtualDeploymentBindingList
		serviceName      string
		serviceNamespace string
	}
	tests := []struct {
		name string
		args args
		want []servicemeshapi.VirtualDeploymentBinding
	}{
		{
			name: "no vdbs present",
			args: args{
				vdbs: &servicemeshapi.VirtualDeploymentBindingList{
					Items: []servicemeshapi.VirtualDeploymentBinding{}},
				serviceName:      "test-service-1",
				serviceNamespace: "test-service-ns-1",
			},
			want: []servicemeshapi.VirtualDeploymentBinding{},
		},
		{
			name: "no vdbs matched",
			args: args{
				vdbs: &servicemeshapi.VirtualDeploymentBindingList{
					Items: []servicemeshapi.VirtualDeploymentBinding{
						vdbVersionV1, vdbVersionV2, vdbVersionV3,
					}},
				serviceName:      "no-vdb-test-service-1-",
				serviceNamespace: "test",
			},
			want: []servicemeshapi.VirtualDeploymentBinding{},
		},
		{
			name: "one vdb matched",
			args: args{
				vdbs: &servicemeshapi.VirtualDeploymentBindingList{
					Items: []servicemeshapi.VirtualDeploymentBinding{
						vdbVersionV1, vdbVersionV2, vdbVersionV3,
					}},
				serviceName:      "test-service",
				serviceNamespace: "test-ns-vdb",
			},
			want: []servicemeshapi.VirtualDeploymentBinding{vdbVersionV1},
		},
		{
			name: "multiple vdbs matched",
			args: args{
				vdbs: &servicemeshapi.VirtualDeploymentBindingList{
					Items: []servicemeshapi.VirtualDeploymentBinding{
						vdbVersionV1, vdbVersionV2, vdbVersionV3,
					}},
				serviceName:      "test-service-2",
				serviceNamespace: "test-ns-vdb",
			},
			want: []servicemeshapi.VirtualDeploymentBinding{vdbVersionV2, vdbVersionV3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterVDBsByServiceRef(tt.args.vdbs, tt.args.serviceName, tt.args.serviceNamespace); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterVDBsByServiceRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
