/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import "strings"

const (
	AuthTypeUserPrincipal     = "user_principal"
	AuthTypeInstancePrincipal = "instance_principal"
	AuthTypeSecurityToken     = "security_token"

	DefaultSecurityTokenConfigFilePath = "/etc/oci/config"
	DefaultSecurityTokenConfigProfile  = "DEFAULT"
)

type UserAuthConfig struct {
	AuthType          string `json:"authType"`
	Region            string `json:"region"`
	Tenancy           string `json:"tenancy"`
	User              string `json:"user"`
	PrivateKey        string `json:"key"`
	Fingerprint       string `json:"fingerprint"`
	Passphrase        string `json:"passphrase"`
	ConfigFilePath    string `json:"configFilePath"`
	ConfigFileProfile string `json:"configFileProfile"`
}

func (u UserAuthConfig) HasAnyUserPrincipalField() bool {
	return strings.TrimSpace(u.Tenancy) != "" ||
		strings.TrimSpace(u.User) != "" ||
		strings.TrimSpace(u.Region) != "" ||
		strings.TrimSpace(u.Fingerprint) != "" ||
		strings.TrimSpace(u.PrivateKey) != ""
}

func (u UserAuthConfig) HasCompleteUserPrincipal() bool {
	return strings.TrimSpace(u.Tenancy) != "" &&
		strings.TrimSpace(u.User) != "" &&
		strings.TrimSpace(u.Region) != "" &&
		strings.TrimSpace(u.Fingerprint) != "" &&
		strings.TrimSpace(u.PrivateKey) != ""
}

func (u UserAuthConfig) WantsSecurityToken() bool {
	return strings.EqualFold(strings.TrimSpace(u.AuthType), AuthTypeSecurityToken)
}
