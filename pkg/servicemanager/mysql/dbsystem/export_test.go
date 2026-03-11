/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
)

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *DbSystemServiceManager, c MySQLDbSystemClientInterface) {
	m.ociClient = c
}

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(dbSystem mysql.DbSystem) (map[string][]byte, error) {
	return getCredentialMap(dbSystem)
}

// ExportDeleteFromSecretForTest exports deleteFromSecret for unit testing.
func ExportDeleteFromSecretForTest(m *DbSystemServiceManager, ctx context.Context, namespace, name string) (bool, error) {
	return m.deleteFromSecret(ctx, namespace, name)
}

// ExportGetDbSystemRetryPredicate returns the shouldRetry predicate from getDbSystemRetryPolicy.
func ExportGetDbSystemRetryPredicate(m *DbSystemServiceManager) func(common.OCIOperationResponse) bool {
	return m.getDbSystemRetryPolicy(1).ShouldRetryOperation
}

// ExportGetDbSystemNextDuration returns the nextDuration function from getDbSystemRetryPolicy.
func ExportGetDbSystemNextDuration(m *DbSystemServiceManager) func(common.OCIOperationResponse) time.Duration {
	return m.getDbSystemRetryPolicy(1).NextDuration
}
