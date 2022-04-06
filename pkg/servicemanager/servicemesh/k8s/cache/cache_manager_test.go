/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cache

import (
	"reflect"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/test/servicemesh/framework"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	api rest.Interface
	f   *framework.Framework
)

func BeforeEach(t *testing.T) {
	f = framework.NewFakeClientFramework(t)
	api = f.K8sClientset.CoreV1().RESTClient()
}

func AfterEach() {
	f.Cleanup()
}

func TestSharedCustomCaches_GetConfigMapByKey(t *testing.T) {
	type fields struct {
		caches map[commons.InformerCacheType]cache.SharedIndexInformer
	}
	type args struct {
		key          string
		valuePresent bool
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantConfigMap *corev1.ConfigMap
		wantErr       string
	}{
		{
			name: "configmap present",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.ConfigMapsCache: newConfigMapCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-namespace/test-name",
				valuePresent: true,
			},
			wantConfigMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-name",
					Namespace: "test-namespace",
				},
			},
			wantErr: "",
		},
		{
			name: "configmap absent",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.ConfigMapsCache: newConfigMapCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-namespace/test-name",
				valuePresent: false,
			},
			wantConfigMap: nil,
			wantErr:       "configMap does not exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BeforeEach(t)

			sc := &SharedCustomCaches{
				caches: tt.fields.caches,
			}
			if tt.args.valuePresent {
				tt.fields.caches[commons.ConfigMapsCache].GetStore().Add(tt.wantConfigMap)
				time.Sleep(time.Second)
				assert.Equal(t, 1, len(sc.caches[commons.ConfigMapsCache].GetStore().List()))
			}

			gotConfigMap, err := sc.GetConfigMapByKey(tt.args.key)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
			} else if !reflect.DeepEqual(gotConfigMap, tt.wantConfigMap) {
				t.Errorf("GetConfigMapByKey() gotConfigMap = %v, want %v", gotConfigMap, tt.wantConfigMap)
			}
			AfterEach()
		})
	}
}

func TestSharedCustomCaches_GetNamespaceByKey(t *testing.T) {
	type fields struct {
		caches map[commons.InformerCacheType]cache.SharedIndexInformer
	}
	type args struct {
		key          string
		valuePresent bool
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantNamespace *corev1.Namespace
		wantErr       string
	}{
		{
			name: "namespace present",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.NamespacesCache: newNamespaceCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-ns",
				valuePresent: true,
			},
			wantNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			wantErr: "",
		},
		{
			name: "namespace absent",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.NamespacesCache: newConfigMapCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-name",
				valuePresent: false,
			},
			wantNamespace: nil,
			wantErr:       "namespace does not exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BeforeEach(t)

			sc := &SharedCustomCaches{
				caches: tt.fields.caches,
			}
			if tt.args.valuePresent {
				tt.fields.caches[commons.NamespacesCache].GetStore().Add(tt.wantNamespace)
				time.Sleep(time.Second)
				assert.Equal(t, 1, len(sc.caches[commons.NamespacesCache].GetStore().List()))
			}

			gotNamespace, err := sc.GetNamespaceByKey(tt.args.key)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
			} else if !reflect.DeepEqual(gotNamespace, tt.wantNamespace) {
				t.Errorf("GetNamespaceByKey() gotNamespace = %v, want %v", gotNamespace, tt.wantNamespace)
			}
			AfterEach()
		})
	}
}

func TestSharedCustomCaches_GetServicesByKey(t *testing.T) {
	type fields struct {
		caches map[commons.InformerCacheType]cache.SharedIndexInformer
	}
	type args struct {
		key          string
		valuePresent bool
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantService *corev1.Service
		wantErr     string
	}{
		{
			name: "service present",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.ServicesCache: newNamespaceCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-service",
				valuePresent: true,
			},
			wantService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-service",
				},
			},
			wantErr: "",
		},
		{
			name: "service absent",
			fields: fields{
				caches: map[commons.InformerCacheType]cache.SharedIndexInformer{
					commons.ServicesCache: newServicesCache(api, time.Millisecond, loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("Cache")}).GetInformer(),
				},
			},
			args: args{
				key:          "test-name",
				valuePresent: false,
			},
			wantService: nil,
			wantErr:     "service does not exists",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BeforeEach(t)

			sc := &SharedCustomCaches{
				caches: tt.fields.caches,
			}
			if tt.args.valuePresent {
				tt.fields.caches[commons.ServicesCache].GetStore().Add(tt.wantService)
				time.Sleep(time.Second)
				assert.Equal(t, 1, len(sc.caches[commons.ServicesCache].GetStore().List()))
			}

			gotService, err := sc.GetServiceByKey(tt.args.key)
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
			} else if !reflect.DeepEqual(gotService, tt.wantService) {
				t.Errorf("GetNamespaceByKey() gotService = %v, want %v", gotService, tt.wantService)
			}
			AfterEach()
		})
	}
}
