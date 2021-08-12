/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

type OsokConfig interface {
	Auth() UserAuthConfig
	UseInstancePrincipals() bool
	VaultDetails() string
}

type osokConfig struct {
	auth                  UserAuthConfig
	useInstancePrincipals bool
	vaultDetails          string
}

var _ OsokConfig = osokConfig{}

func (o osokConfig) Auth() UserAuthConfig {
	return o.auth
}

func (o osokConfig) UseInstancePrincipals() bool {
	return o.useInstancePrincipals
}

func (o osokConfig) VaultDetails() string {
	return o.vaultDetails
}
