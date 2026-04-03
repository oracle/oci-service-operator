//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage

import (
	"context"
	"fmt"

	"github.com/oracle/oci-service-operator/pkg/servicemanager"
)

func (m *ObjectStorageBucketServiceManager) addToSecret(ctx context.Context, k8sNamespace string, resourceName string,
	namespace string, bucketName string) (bool, error) {

	m.Log.InfoLog("Creating the ObjectStorageBucket connection secret")
	region := ""
	if m.Provider != nil {
		if providerRegion, err := m.Provider.Region(); err == nil {
			region = providerRegion
		}
	}
	credMap := getCredentialMap(namespace, bucketName, region)

	m.Log.InfoLog(fmt.Sprintf("Creating secret for ObjectStorageBucket %s in namespace %s", resourceName, k8sNamespace))
	return servicemanager.EnsureOwnedSecret(ctx, m.CredentialClient, resourceName, k8sNamespace, "ObjectStorageBucket", resourceName, credMap)
}

func getCredentialMap(namespace, bucketName string, region string) map[string][]byte {
	host := "objectstorage.oraclecloud.com"
	if region != "" {
		host = fmt.Sprintf("objectstorage.%s.oraclecloud.com", region)
	}
	endpoint := fmt.Sprintf("https://%s/n/%s/b/%s", host, namespace, bucketName)
	return map[string][]byte{
		"namespace":   []byte(namespace),
		"bucketName":  []byte(bucketName),
		"apiEndpoint": []byte(endpoint),
	}
}
