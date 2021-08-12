/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v41/streaming"
)

func (c *StreamServiceManager) addToSecret(ctx context.Context, namespace string, streamName string,
	stream streaming.Stream) (bool, error) {

	c.Log.InfoLog("Creating the Credential Map")
	credMap, err := getCredentialMap(stream)
	if err != nil {
		c.Log.ErrorLog(err, "Error while creating Stream secret map")
		return false, err
	}

	c.Log.InfoLog("Creating the Stream MessageEndpoint secret")
	c.Log.InfoLog(fmt.Sprintf("Received information for secret creation - namespace: %s streamName: %s ", namespace, streamName))
	return c.CredentialClient.CreateSecret(ctx, streamName, namespace, nil, credMap)
}

func getCredentialMap(resp streaming.Stream) (map[string][]byte, error) {
	credMap := make(map[string][]byte)
	credMap["endpoint"] = []byte(*resp.MessagesEndpoint)
	return credMap, nil
}

func (c *StreamServiceManager) deleteFromSecret(ctx context.Context, namespace string, streamName string) (bool, error) {
	c.Log.InfoLog(fmt.Sprintf("Received information for secret deletion - namespace: %s streamName: %s ", namespace, streamName))
	return c.CredentialClient.DeleteSecret(ctx, streamName, namespace)
}
