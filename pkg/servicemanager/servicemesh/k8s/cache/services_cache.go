/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package cache

import (
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ServicesCache struct {
	api          rest.Interface
	resyncPeriod time.Duration
	log          loggerutil.OSOKLogger
}

//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get

func (a *ServicesCache) GetInformer() cache.SharedIndexInformer {
	watchlist := cache.NewListWatchFromClient(a.api, "services", apiv1.NamespaceAll, fields.Everything())
	informer := cache.NewSharedIndexInformer(watchlist, &apiv1.Service{}, a.resyncPeriod, cache.Indexers{})
	return informer
}

func newServicesCache(api rest.Interface, resyncPeriod time.Duration, log loggerutil.OSOKLogger) *ServicesCache {
	return &ServicesCache{
		api:          api,
		resyncPeriod: resyncPeriod,
		log:          log,
	}
}
