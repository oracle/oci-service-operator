//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch

// SetClientForTest injects a fake OCI client into the service manager for unit testing.
func SetClientForTest(mgr *OpenSearchClusterServiceManager, client OpensearchClusterClientInterface) {
	mgr.ociClient = client
}
