/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cache

import (
	"time"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type NamespaceCache struct {
	api          rest.Interface
	resyncPeriod time.Duration
	log          loggerutil.OSOKLogger
}

func (a *NamespaceCache) GetInformer() cache.SharedIndexInformer {
	watchlist := cache.NewListWatchFromClient(a.api, "nameSpaces", corev1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedIndexInformer(watchlist, &corev1.Namespace{}, a.resyncPeriod, cache.Indexers{})
	return informer
}

func newNamespaceCache(api rest.Interface, resyncPeriod time.Duration, log loggerutil.OSOKLogger) *NamespaceCache {
	return &NamespaceCache{
		api:          api,
		resyncPeriod: resyncPeriod,
		log:          log,
	}
}
