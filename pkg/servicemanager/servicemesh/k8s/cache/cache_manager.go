/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cache

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

/**
  Given a customConfig caches are created to reduce API calls to k8s server.
  Currently we are watching configs, namespaces and services
*/

type CustomCacheConfig struct {
	ResyncPeriod time.Duration
	ClientSet    kubernetes.Interface
	Log          loggerutil.OSOKLogger
}

type CacheMap interface {
	NewSharedCaches() map[commons.InformerCacheType]cache.SharedIndexInformer
}

// This abstraction will enable to create caches which can be bundles together and passed around
type CustomCache interface {
	GetInformer() cache.SharedIndexInformer
}
type SharedCustomCaches struct {
	caches map[commons.InformerCacheType]cache.SharedIndexInformer
}

type CacheSetup interface {
	SetupWithManager(mgr ctrl.Manager, cache cache.SharedInformer)
}

type CacheMapClient interface {
	GetConfigMapByKey(key string) (item *corev1.ConfigMap, err error)
	GetNamespaceByKey(key string) (item *corev1.Namespace, err error)
	GetServiceByKey(key string) (item *corev1.Service, err error)
}

func (cacheConfig *CustomCacheConfig) NewSharedCaches() *SharedCustomCaches {
	api := cacheConfig.ClientSet.CoreV1().RESTClient()
	cacheMap := map[commons.InformerCacheType]cache.SharedIndexInformer{
		commons.ConfigMapsCache: newConfigMapCache(api, cacheConfig.ResyncPeriod, cacheConfig.Log).GetInformer(),
		commons.NamespacesCache: newNamespaceCache(api, cacheConfig.ResyncPeriod, cacheConfig.Log).GetInformer(),
		commons.ServicesCache:   newServicesCache(api, cacheConfig.ResyncPeriod, cacheConfig.Log).GetInformer(),
	}
	return &SharedCustomCaches{caches: cacheMap}
}

func (sc *SharedCustomCaches) GetConfigMapByKey(key string) (configMap *corev1.ConfigMap, err error) {
	store := sc.caches[commons.ConfigMapsCache].GetStore()
	item, exists, err := store.GetByKey(key)

	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("configMap does not exists")
	}
	configMap, ok := item.(*corev1.ConfigMap)
	if !ok {
		return nil, errors.New("unknown type")
	}

	return configMap, err

}
func (sc *SharedCustomCaches) GetServiceByKey(key string) (service *corev1.Service, err error) {
	store := sc.caches[commons.ServicesCache].GetStore()
	item, exists, err := store.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("service does not exists")
	}
	service, ok := item.(*corev1.Service)
	if !ok {
		return nil, errors.New("unknown type")
	}
	return service, nil

}
func (sc *SharedCustomCaches) GetNamespaceByKey(key string) (namespace *corev1.Namespace, err error) {
	store := sc.caches[commons.NamespacesCache].GetStore()
	item, exists, err := store.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("namespace does not exists")
	}
	namespace, ok := item.(*corev1.Namespace)
	if !ok {
		return nil, errors.New("unknown type")
	}
	return namespace, nil

}

func (sc *SharedCustomCaches) SetupWithManager(mgr ctrl.Manager, log loggerutil.OSOKLogger) {
	for _, cache := range sc.caches {
		cache := cache
		err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			go cache.Run(ctx.Done())
			return nil
		}))
		if err != nil {
			log.Logger.Error(err, "unable to create caches")
			os.Exit(1)
		}

	}
}
