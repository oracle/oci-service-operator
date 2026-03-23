/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"

	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	newMySqlDbSystemServiceClient = func(manager *MySqlDbSystemServiceManager) MySqlDbSystemServiceClient {
		return legacyMySqlDbSystemServiceClient{
			delegate: NewDbSystemServiceManagerWithDeps(servicemanager.RuntimeDeps{
				Provider:         manager.Provider,
				CredentialClient: manager.CredentialClient,
				Scheme:           manager.Scheme,
				Log:              manager.Log,
				Metrics:          manager.Metrics,
			}),
		}
	}
}

type legacyMySqlDbSystemServiceClient struct {
	delegate *DbSystemServiceManager
}

var _ MySqlDbSystemServiceClient = legacyMySqlDbSystemServiceClient{}

func (c legacyMySqlDbSystemServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *mysqlv1beta1.MySqlDbSystem,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c legacyMySqlDbSystemServiceClient) Delete(
	ctx context.Context,
	resource *mysqlv1beta1.MySqlDbSystem,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
