/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"

	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	newAutonomousDatabasesServiceClient = func(manager *AutonomousDatabasesServiceManager) AutonomousDatabasesServiceClient {
		return legacyAutonomousDatabasesServiceClient{
			delegate: NewAdbServiceManagerWithDeps(servicemanager.RuntimeDeps{
				Provider:         manager.Provider,
				CredentialClient: manager.CredentialClient,
				Scheme:           manager.Scheme,
				Log:              manager.Log,
				Metrics:          manager.Metrics,
			}),
		}
	}
}

type legacyAutonomousDatabasesServiceClient struct {
	delegate *AdbServiceManager
}

var _ AutonomousDatabasesServiceClient = legacyAutonomousDatabasesServiceClient{}

func (c legacyAutonomousDatabasesServiceClient) CreateOrUpdate(
	ctx context.Context,
	resource *databasev1beta1.AutonomousDatabases,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	return c.delegate.CreateOrUpdate(ctx, resource, req)
}

func (c legacyAutonomousDatabasesServiceClient) Delete(
	ctx context.Context,
	resource *databasev1beta1.AutonomousDatabases,
) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
