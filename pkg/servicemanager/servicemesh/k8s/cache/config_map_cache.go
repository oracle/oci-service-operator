/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cache

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

type ConfigMapCache struct {
	api          rest.Interface
	resyncPeriod time.Duration
	log          loggerutil.OSOKLogger
	namespace    string
}

func (a *ConfigMapCache) GetInformer() cache.SharedIndexInformer {
	watchlist := cache.NewListWatchFromClient(a.api, "configMaps", a.namespace,
		fields.Everything())
	informer := cache.NewSharedIndexInformer(watchlist, &corev1.ConfigMap{}, a.resyncPeriod, cache.Indexers{})
	return informer
}

func newConfigMapCache(api rest.Interface, resyncPeriod time.Duration, log loggerutil.OSOKLogger) *ConfigMapCache {
	return &ConfigMapCache{
		api:          api,
		resyncPeriod: resyncPeriod,
		log:          log,
	}
}
