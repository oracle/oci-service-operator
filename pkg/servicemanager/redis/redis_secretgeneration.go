//go:build legacyservicemanager

/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
)

func (c *RedisClusterServiceManager) addToSecret(ctx context.Context, namespace string, clusterName string,
	cluster redis.RedisCluster) (bool, error) {

	c.Log.InfoLog("Creating the RedisCluster connection secret")
	credMap := getCredentialMap(cluster)

	c.Log.InfoLog(fmt.Sprintf("Creating secret for RedisCluster %s in namespace %s", clusterName, namespace))
	return servicemanager.EnsureOwnedSecret(ctx, c.CredentialClient, clusterName, namespace, "RedisCluster", clusterName, credMap)
}

func getCredentialMap(cluster redis.RedisCluster) map[string][]byte {
	credMap := make(map[string][]byte)

	if cluster.PrimaryFqdn != nil {
		credMap["primaryFqdn"] = []byte(*cluster.PrimaryFqdn)
	}
	if cluster.PrimaryEndpointIpAddress != nil {
		credMap["primaryEndpointIpAddress"] = []byte(*cluster.PrimaryEndpointIpAddress)
	}
	if cluster.ReplicasFqdn != nil {
		credMap["replicasFqdn"] = []byte(*cluster.ReplicasFqdn)
	}
	if cluster.ReplicasEndpointIpAddress != nil {
		credMap["replicasEndpointIpAddress"] = []byte(*cluster.ReplicasEndpointIpAddress)
	}

	return credMap
}
