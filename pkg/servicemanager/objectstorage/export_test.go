//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(namespace, bucketName string) map[string][]byte {
	return getCredentialMap(namespace, bucketName, "")
}

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *ObjectStorageBucketServiceManager, c ObjectStorageClientInterface) {
	m.ociClient = c
}
