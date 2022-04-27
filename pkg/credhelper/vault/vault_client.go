/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/pkg/errors"
)

type VaultClient struct {
	Provider common.ConfigurationProvider
	Log      logr.Logger
	KeyId    string
	VaultId  string
}

func NewVaultClient(provider common.ConfigurationProvider, log logr.Logger, keyId string, vaultId string) *VaultClient {
	return &VaultClient{
		Provider: provider,
		Log:      log,
		KeyId:    keyId,
		VaultId:  vaultId,
	}
}

func getVaultsClient(provider common.ConfigurationProvider) (vault.VaultsClient, error) {
	vaultsClient, err := vault.NewVaultsClientWithConfigurationProvider(provider)
	if err != nil {
		return vaultsClient, errors.Wrap(err, "Error initializing the Vaults Client")
	}
	return vaultsClient, nil
}

func (v *VaultClient) CreateSecret(ctx context.Context, secretName string, secretNamespace string, labels map[string]string,
	data map[string][]byte) (bool, error) {
	vaultsClient, err := getVaultsClient(v.Provider)
	if err != nil {
		return false, err
	}

	secretData, err := json.Marshal(data)
	if err != nil {
		return false, errors.Wrapf(err, "Error occured while converting the data to json string")
	}

	base64Str := base64.StdEncoding.EncodeToString(secretData)
	secret := vault.Base64SecretContentDetails{
		Content: &base64Str,
		Stage:   "",
	}

	secretDetails := vault.CreateSecretDetails{
		SecretContent: secret,
		SecretName:    &secretName,
		VaultId:       &v.VaultId,
		Description:   &secretName,
		FreeformTags:  labels,
		KeyId:         &v.KeyId,
	}

	_, err = vaultsClient.CreateSecret(ctx, vault.CreateSecretRequest{
		CreateSecretDetails: secretDetails,
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

func (v *VaultClient) DeleteSecret(ctx context.Context, secretName string, secretNamespace string) (bool, error) {

	return true, nil
}

func (v *VaultClient) GetSecret(ctx context.Context, secretName string, secretNamespace string) (map[string][]byte, error) {
	return nil, nil
}
