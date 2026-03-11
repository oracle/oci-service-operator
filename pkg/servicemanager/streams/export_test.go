/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
)

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *StreamServiceManager, c StreamAdminClientInterface) {
	m.ociClient = c
}

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(stream streaming.Stream) (map[string][]byte, error) {
	return getCredentialMap(stream)
}

// ExportGetStreamRetryPredicate returns the shouldRetry predicate from getStreamRetryPolicy.
func ExportGetStreamRetryPredicate(m *StreamServiceManager) func(common.OCIOperationResponse) bool {
	return m.getStreamRetryPolicy(1).ShouldRetryOperation
}

// ExportDeleteStreamRetryPredicate returns the shouldRetry predicate from deleteStreamRetryPolicy.
func ExportDeleteStreamRetryPredicate(m *StreamServiceManager) func(common.OCIOperationResponse) bool {
	return m.deleteStreamRetryPolicy(1).ShouldRetryOperation
}

// ExportGetStreamNextDuration returns the nextDuration function from getStreamRetryPolicy.
func ExportGetStreamNextDuration(m *StreamServiceManager) func(common.OCIOperationResponse) time.Duration {
	return m.getStreamRetryPolicy(1).NextDuration
}

// ExportDeleteStreamNextDuration returns the nextDuration function from deleteStreamRetryPolicy.
func ExportDeleteStreamNextDuration(m *StreamServiceManager) func(common.OCIOperationResponse) time.Duration {
	return m.deleteStreamRetryPolicy(1).NextDuration
}
