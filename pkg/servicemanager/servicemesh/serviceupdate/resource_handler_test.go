/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package serviceupdate

import (
	"context"
	"testing"
	"time"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	merrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"
	. "github.com/oracle/oci-service-operator/test/servicemesh/k8s"
	testProxyController "github.com/oracle/oci-service-operator/test/servicemesh/proxycontroller"
)

func Test_defaultResourceHandler_FetchNamespaceInjectionLabel(t *testing.T) {
	type args struct {
		req                     ctrl.Request
		namespaceInjectionLabel string
		namespace               *corev1.Namespace
	}
	type want struct {
		err                     error
		found                   bool
		namespaceInjectionLabel string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "namespace not found",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "my-namespace",
					},
				},
				namespaceInjectionLabel: "",
			},
			want: want{
				err:                     nil,
				found:                   false,
				namespaceInjectionLabel: "",
			},
		},
		{
			name: "namespace does not have label",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "my-namespace",
					},
				},
				namespaceInjectionLabel: "",
				namespace: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-namespace",
					},
				},
			},
			want: want{
				err:                     nil,
				found:                   false,
				namespaceInjectionLabel: "",
			},
		},
		{
			name: "namespace has label",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: "my-namespace",
					},
				},
				namespaceInjectionLabel: "",
				namespace: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-namespace",
						Labels: map[string]string{
							commons.ProxyInjectionLabel: "my-label",
						},
					},
				},
			},
			want: want{
				err:                     nil,
				found:                   true,
				namespaceInjectionLabel: "my-label",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = api.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClientBuilder := testclient.NewClientBuilder().WithScheme(k8sSchema)
			if tt.args.namespace != nil {
				k8sClientBuilder = k8sClientBuilder.WithObjects(tt.args.namespace)
			}
			k8sClient := k8sClientBuilder.Build()

			m := &defaultResourceHandler{
				k8sClient: k8sClient,
			}

			found, err := m.FetchNamespaceInjectionLabel(ctx, tt.args.req, &tt.args.namespaceInjectionLabel)
			assert.Equal(t, tt.want.found, found)
			if tt.want.err != nil {
				assert.EqualError(t, err, tt.want.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.namespaceInjectionLabel, tt.args.namespaceInjectionLabel)
			}
		})
	}
}

func Test_defaultResourceHandler_FetchService(t *testing.T) {
	type args struct {
		namespacedName types.NamespacedName
		service        *corev1.Service
	}
	type want struct {
		err     error
		service *corev1.Service
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "service not found",
			args: args{
				namespacedName: types.NamespacedName{
					Name:      "my-service",
					Namespace: "my-namespace",
				},
				service: nil,
			},
			want: want{
				err:     merrors.NewDoNotRequeue(),
				service: nil,
			},
		},
		{
			name: "service is deleted",
			args: args{
				namespacedName: types.NamespacedName{
					Name:      "my-service",
					Namespace: "my-namespace",
				},
				service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "my-service",
						Namespace:         "my-namespace",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				},
			},
			want: want{
				err:     merrors.NewDoNotRequeue(),
				service: nil,
			},
		},
		{
			name: "service found",
			args: args{
				namespacedName: types.NamespacedName{
					Name:      "my-service",
					Namespace: "my-namespace",
				},
				service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-service",
						Namespace: "my-namespace",
					},
				},
			},
			want: want{
				err: nil,
				service: &corev1.Service{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Service",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-service",
						Namespace: "my-namespace",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			err = api.AddToScheme(k8sSchema)
			assert.NoError(t, err)
			k8sClientBuilder := testclient.NewClientBuilder().WithScheme(k8sSchema)
			if tt.args.service != nil {
				k8sClientBuilder = k8sClientBuilder.WithObjects(tt.args.service)
			}
			k8sClient := k8sClientBuilder.Build()

			m := &defaultResourceHandler{
				k8sClient: k8sClient,
			}

			service, err := m.FetchService(ctx, tt.args.namespacedName)
			if tt.want.err != nil {
				assert.EqualError(t, err, tt.want.err.Error())
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				opts := cmpopts.IgnoreTypes((*metav1.Time)(nil))
				assert.True(t, cmp.Equal(tt.want.service, service, opts), "diff", cmp.Diff(tt.want.service, service, opts))
			}
		})
	}
}

func Test_defaultResourceHandler_evictPods(t *testing.T) {
	type args struct {
		namespaceInjectionLabel string
		testControllerArgs      testProxyController.ProxyControllerArgs
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "pod is evicted",
			args: args{
				testControllerArgs: testProxyController.ProxyControllerArgs{
					NamespaceLabel:      commons.Enabled,
					VDBRefPodAnnotation: "test/product-vdb",
					UpdateConfigMap:     true,
					CreateService:       true,
					CreateVDB:           true,
					CreatePDB:           false,
				},
				namespaceInjectionLabel: "enabled",
			},
			wantErr: nil,
		},
		{
			name: "pod is not evicted",
			args: args{
				testControllerArgs: testProxyController.ProxyControllerArgs{
					NamespaceLabel:      commons.Enabled,
					VDBRefPodAnnotation: "test/product-vdb",
					UpdateConfigMap:     true,
					CreateService:       true,
					CreateVDB:           true,
					CreatePDB:           true,
				},
				namespaceInjectionLabel: "enabled",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testFramework := framework.NewTestEnvClientFramework(t)
			testProxyController.Initialize(true)
			err := CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: testProxyController.TestNamespace.Name}, testProxyController.TestNamespace)
			if err != nil {
				t.Fatal("Failed to create test namespace", err)
			}
			err = CreatePod(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testProxyController.TestPod.Namespace, Name: testProxyController.TestPod.Name}, testProxyController.TestPod)
			if err != nil {
				t.Fatal("Failed to create test pod", err)
			}
			err = CreateNamespace(ctx, testFramework.K8sClient, types.NamespacedName{Name: commons.OsokNamespace}, testProxyController.MeshOperatorNamespace)
			if err != nil {
				t.Fatal("Failed to create mesh operator namespace", err)
			}
			err = CreateConfigMap(ctx, testFramework.K8sClient, types.NamespacedName{Namespace: testProxyController.MeshConfigMap.Namespace, Name: testProxyController.MeshConfigMap.Name}, testProxyController.MeshConfigMap)
			if err != nil {
				t.Fatal("Failed to create test config map", err)
			}

			m := &defaultResourceHandler{
				k8sClient: testFramework.K8sClient,
				clientSet: testFramework.K8sClientset,
				log:       loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName("ServiceUpdateHandler")},
			}

			testProxyController.UpdateResources(ctx, testFramework, tt.args.testControllerArgs, t)

			pods := &corev1.PodList{
				Items: []corev1.Pod{
					*testProxyController.TestPod,
				},
			}
			vdbs := []servicemeshapi.VirtualDeploymentBinding{
				*testProxyController.TestVDB,
			}
			err = m.evictPods(ctx, pods, vdbs, tt.args.namespaceInjectionLabel)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
