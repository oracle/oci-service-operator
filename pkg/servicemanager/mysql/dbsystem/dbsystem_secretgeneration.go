/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	"strconv"
)

func (c *DbSystemServiceManager) addToSecret(ctx context.Context, namespace string, dbSystemName string,
	mysqlDbSystem mysql.DbSystem) (bool, error) {

	c.Log.InfoLog("Creating the Credential Map")
	credMap, err := getCredentialMap(mysqlDbSystem)
	if err != nil {
		c.Log.ErrorLog(err, "Error while creating Mysql secret map")
		return false, err
	}

	c.Log.InfoLog("Creating the MySql DbSystem Endpoint secret")
	c.Log.InfoLog(fmt.Sprintf("Received information for secret creation - Namespace : %s MysqlDbSystem name: %s ", namespace, dbSystemName))
	return c.CredentialClient.CreateSecret(ctx, dbSystemName, namespace, nil, credMap)
}

func getCredentialMap(resp mysql.DbSystem) (map[string][]byte, error) {

	credMap := make(map[string][]byte)

	credMap["PrivateIPAddress"] = []byte(*resp.IpAddress)
	if resp.HostnameLabel != nil {
		credMap["InternalFQDN"] = []byte(*resp.HostnameLabel)
	} else {
		credMap["InternalFQDN"] = []byte("")
	}
	credMap["AvailabilityDomain"] = []byte(*resp.AvailabilityDomain)
	credMap["FaultDomain"] = []byte(*resp.FaultDomain)

	credMap["MySQLPort"] = []byte(strconv.Itoa(*resp.Port))
	credMap["MySQLXProtocolPort"] = []byte(strconv.Itoa(*resp.PortX))
	reqBodyBytes := new(bytes.Buffer)
	err := json.NewEncoder(reqBodyBytes).Encode(resp.Endpoints)
	if err != nil {
		fmt.Sprintln("Unexpected parsing error")
	}
	credMap["Endpoints"] = reqBodyBytes.Bytes()

	return credMap, nil
}

func (c *DbSystemServiceManager) deleteFromSecret(ctx context.Context, namespace string, dbSystemName string) (bool, error) {
	c.Log.InfoLog(fmt.Sprintf("Received information for secret deletion - Namespace: %s MysqlDbSystem: %s ", namespace, dbSystemName))
	return c.CredentialClient.DeleteSecret(ctx, dbSystemName, namespace)
}
