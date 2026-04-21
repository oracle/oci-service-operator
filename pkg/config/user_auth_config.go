/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import "strings"

const (
	AuthTypeUserPrincipal                    = "user_principal"
	AuthTypeInstancePrincipal                = "instance_principal"
	AuthTypeInstancePrincipalWithCerts       = "instance_principal_with_certs"
	AuthTypeInstancePrincipalDelegationToken = "instance_principal_delegation_token"
	AuthTypeResourcePrincipal                = "resource_principal"
	AuthTypeResourcePrincipalDelegationToken = "resource_principal_delegation_token"
	AuthTypeOKEWorkloadIdentity              = "oke_workload_identity"
	AuthTypeSecurityToken                    = "security_token"
	AuthTypeWorkloadIdentityFederation       = "workload_identity_federation"
	AuthTypeOAuthDelegationToken             = "oauth_delegation_token"

	sdkAuthTypeInstancePrincipalDelegationToken = "instance_principle_delegation_token"
	sdkAuthTypeResourcePrincipalDelegationToken = "resource_principle_delegation_token"

	DefaultOCIConfigFilePath = "/etc/oci/config"
	DefaultOCIConfigProfile  = "DEFAULT"

	DefaultSecurityTokenConfigFilePath = DefaultOCIConfigFilePath
	DefaultSecurityTokenConfigProfile  = DefaultOCIConfigProfile
)

// UserAuthConfig carries auth-mode selection plus any mode-specific inputs.
// The name is kept for backward compatibility with existing callers.
type UserAuthConfig struct {
	AuthType                                  string `json:"authType"`
	Region                                    string `json:"region"`
	Tenancy                                   string `json:"tenancy"`
	User                                      string `json:"user"`
	PrivateKey                                string `json:"key"`
	Fingerprint                               string `json:"fingerprint"`
	Passphrase                                string `json:"passphrase"`
	ConfigFilePath                            string `json:"configFilePath"`
	ConfigFileProfile                         string `json:"configFileProfile"`
	InstancePrincipalDelegationToken          string `json:"instancePrincipalDelegationToken"`
	ResourcePrincipalDelegationToken          string `json:"resourcePrincipalDelegationToken"`
	InstancePrincipalLeafCertificatePath      string `json:"instancePrincipalLeafCertificatePath"`
	InstancePrincipalLeafPrivateKeyPath       string `json:"instancePrincipalLeafPrivateKeyPath"`
	InstancePrincipalLeafPrivateKeyPassphrase string `json:"instancePrincipalLeafPrivateKeyPassphrase"`
	InstancePrincipalIntermediateCertPathList string `json:"instancePrincipalIntermediateCertPathList"`
}

func (u UserAuthConfig) HasAnyUserPrincipalField() bool {
	return strings.TrimSpace(u.Tenancy) != "" ||
		strings.TrimSpace(u.User) != "" ||
		strings.TrimSpace(u.Region) != "" ||
		strings.TrimSpace(u.Fingerprint) != "" ||
		strings.TrimSpace(u.PrivateKey) != ""
}

func (u UserAuthConfig) HasAnyUserPrincipalInput() bool {
	return u.HasAnyUserPrincipalField() || u.HasConfigFileReference()
}

func (u UserAuthConfig) HasCompleteUserPrincipal() bool {
	return strings.TrimSpace(u.Tenancy) != "" &&
		strings.TrimSpace(u.User) != "" &&
		strings.TrimSpace(u.Region) != "" &&
		strings.TrimSpace(u.Fingerprint) != "" &&
		strings.TrimSpace(u.PrivateKey) != ""
}

func (u UserAuthConfig) HasConfigFileReference() bool {
	return strings.TrimSpace(u.ConfigFilePath) != "" ||
		strings.TrimSpace(u.ConfigFileProfile) != ""
}

func (u UserAuthConfig) EffectiveConfigFilePath() string {
	configFilePath := strings.TrimSpace(u.ConfigFilePath)
	if configFilePath == "" {
		return DefaultOCIConfigFilePath
	}
	return configFilePath
}

func (u UserAuthConfig) EffectiveConfigFileProfile() string {
	profile := strings.TrimSpace(u.ConfigFileProfile)
	if profile == "" {
		return DefaultOCIConfigProfile
	}
	return profile
}

func (u UserAuthConfig) HasInstancePrincipalWithCertsInputs() bool {
	return strings.TrimSpace(u.InstancePrincipalLeafCertificatePath) != "" ||
		strings.TrimSpace(u.InstancePrincipalLeafPrivateKeyPath) != "" ||
		strings.TrimSpace(u.InstancePrincipalIntermediateCertPathList) != ""
}

func (u UserAuthConfig) IntermediateCertificatePaths() []string {
	raw := strings.TrimSpace(u.InstancePrincipalIntermediateCertPathList)
	if raw == "" {
		return nil
	}

	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	paths := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		paths = append(paths, field)
	}
	return paths
}

func (u UserAuthConfig) NormalizedAuthType() string {
	switch strings.ToLower(strings.TrimSpace(u.AuthType)) {
	case sdkAuthTypeInstancePrincipalDelegationToken:
		return AuthTypeInstancePrincipalDelegationToken
	case sdkAuthTypeResourcePrincipalDelegationToken:
		return AuthTypeResourcePrincipalDelegationToken
	default:
		return strings.ToLower(strings.TrimSpace(u.AuthType))
	}
}

func (u UserAuthConfig) EffectiveAuthType() string {
	if authType := u.NormalizedAuthType(); authType != "" {
		return authType
	}
	if u.HasAnyUserPrincipalInput() {
		return AuthTypeUserPrincipal
	}
	return AuthTypeInstancePrincipal
}

func (u UserAuthConfig) WantsSecurityToken() bool {
	return u.EffectiveAuthType() == AuthTypeSecurityToken
}
