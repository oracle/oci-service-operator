/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
)

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *AdbServiceManager, c DatabaseClientInterface) {
	m.ociClient = c
}

// ExportAdbRetryPredicate returns the shouldRetry predicate from getAdbRetryPolicy.
func ExportAdbRetryPredicate(m *AdbServiceManager) func(common.OCIOperationResponse) bool {
	return m.getAdbRetryPolicy(1).ShouldRetryOperation
}

// ExportAdbRetryNextDuration returns the nextDuration function from getAdbRetryPolicy.
func ExportAdbRetryNextDuration(m *AdbServiceManager) func(common.OCIOperationResponse) time.Duration {
	return m.getAdbRetryPolicy(1).NextDuration
}

// ExportExponentialBackoffPredicate returns the shouldRetry predicate from getExponentialBackoffRetryPolicy.
func ExportExponentialBackoffPredicate(m *AdbServiceManager) func(common.OCIOperationResponse) bool {
	return m.getExponentialBackoffRetryPolicy(1).ShouldRetryOperation
}

// ExportExponentialBackoffNextDuration returns the nextDuration from getExponentialBackoffRetryPolicy.
func ExportExponentialBackoffNextDuration(m *AdbServiceManager) func(common.OCIOperationResponse) time.Duration {
	return m.getExponentialBackoffRetryPolicy(1).NextDuration
}

// ExportGetCredentialMapForTest exports getCredentialMap for unit testing.
func ExportGetCredentialMapForTest(adbDisplayName string, resp database.GenerateAutonomousDatabaseWalletResponse) (map[string][]byte, error) {
	return getCredentialMap(adbDisplayName, resp)
}
