/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import "github.com/oracle/oci-go-sdk/v65/common"

type UserPrincipal struct {
	UserId      string `json:"userId"`
	Tenancy     string `json:"tenancy"`
	Region      string `json:"region"`
	Fingerprint string `json:"fingerprint"`
	PrivateKey  string `json:"privateKey"`
	Passphrase  string `json:"passphrase"`
}

func (up UserPrincipal) GetAuthProvider() common.ConfigurationProvider {
	return common.NewRawConfigurationProvider(up.Tenancy, up.UserId, up.Region, up.Fingerprint, up.PrivateKey, &up.Passphrase)
}
