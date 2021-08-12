/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"github.com/oracle/oci-go-sdk/v41/common"
	"github.com/oracle/oci-go-sdk/v41/database"
	"io"
	"io/ioutil"
	"math"
	"time"
)

func (c *AdbServiceManager) GenerateWallet(ctx context.Context, adbId string, adbDisplayName string,
	walletSecretName string, namespace string, walletName string, adbInstanceName string) (bool, error) {

	dbClient := getDbClient(c.Provider)

	if walletName == "" {
		c.Log.DebugLog("Autonomous Database wallet password name was not provided. Setting default")
		walletName = fmt.Sprintf("%s-wallet", adbInstanceName)
	}

	c.Log.InfoLog("Checking if the wallet secret already exists")
	_, err := c.CredentialClient.GetSecret(ctx, walletName, namespace)
	if err == nil {
		c.Log.InfoLog("Wallet already exists. Not generating wallet.")
		return true, nil
	}

	c.Log.InfoLog("Getting the wallet password from the secret")
	pwd, err := c.getWalletPassword(ctx, walletSecretName, namespace)
	if err != nil {
		return false, err
	}

	retryPolicy := c.getExponentialBackoffRetryPolicy(8)

	req := database.GenerateAutonomousDatabaseWalletRequest{
		AutonomousDatabaseId: &adbId,
		GenerateAutonomousDatabaseWalletDetails: database.GenerateAutonomousDatabaseWalletDetails{
			Password: pwd,
		},
		RequestMetadata: common.RequestMetadata{RetryPolicy: &retryPolicy},
	}

	c.Log.InfoLog("Generating the Autonomous Database Wallet")
	resp, err := dbClient.GenerateAutonomousDatabaseWallet(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, fmt.Sprintf("Error while generating wallet for Autonomous Database %s", adbDisplayName))
		return false, err
	}

	c.Log.InfoLog("Creating the Credential Map")
	credMap, err := getCredentialMap(adbDisplayName, resp)
	if err != nil {
		c.Log.ErrorLog(err, "Error while creating wallet map")
		return false, err
	}

	c.Log.InfoLog("Creating the Wallet secret")
	return c.CredentialClient.CreateSecret(ctx, walletName, namespace, nil, credMap)
}

func getCredentialMap(adbDisplayName string, resp database.GenerateAutonomousDatabaseWalletResponse) (map[string][]byte, error) {
	credMap := make(map[string][]byte)

	tempZip, err := ioutil.TempFile("", fmt.Sprintf("%s-wallet*.zip", adbDisplayName))
	if err != nil {
		return nil, err
	}
	defer tempZip.Close()

	if _, err := io.Copy(tempZip, resp.Content); err != nil {
		return nil, err
	}

	reader, err := zip.OpenReader(tempZip.Name())
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	for _, file := range reader.File {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}

		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		credMap[file.Name] = content
	}

	return credMap, nil
}

func (c *AdbServiceManager) getWalletPassword(ctx context.Context, secretName string, ns string) (*string, error) {
	secretMap, err := c.CredentialClient.GetSecret(ctx, secretName, ns)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the wallet password secret")
		return nil, err
	}

	pwd, ok := secretMap["walletPassword"]
	if !ok {
		c.Log.ErrorLog(err, "password key 'walletPassword' in wallet password secret is not found")
		return nil, errors.New("password key 'walletPassword' in wallet password secret is not found")
	}

	pwdString := string(pwd)
	return &pwdString, nil
}

func (c *AdbServiceManager) getExponentialBackoffRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		return !(response.Error == nil && response.Response.HTTPResponse().StatusCode >= 200 &&
			response.Response.HTTPResponse().StatusCode < 300)
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
