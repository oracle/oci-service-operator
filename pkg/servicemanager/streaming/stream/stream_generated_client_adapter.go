/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package stream

import (
	"context"

	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	legacystreams "github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	newStreamServiceClient = func(manager *StreamServiceManager) StreamServiceClient {
		return legacyStreamServiceClient{
			delegate: legacystreams.NewStreamServiceManagerWithDeps(servicemanager.RuntimeDeps{
				Provider:         manager.Provider,
				CredentialClient: manager.CredentialClient,
				Scheme:           manager.Scheme,
				Log:              manager.Log,
				Metrics:          manager.Metrics,
			}),
		}
	}
}

type legacyStreamServiceClient struct {
	delegate *legacystreams.StreamServiceManager
}

var _ StreamServiceClient = legacyStreamServiceClient{}

func (c legacyStreamServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c legacyStreamServiceClient) Delete(
	ctx context.Context,
	resource *streamingv1beta1.Stream,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
