/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"os"
	"strconv"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

var (
	configDetails osokConfig
)

func GetConfigDetails(log loggerutil.OSOKLogger) osokConfig {
	configDetails = osokConfig{}

	ip := os.Getenv("USEINSTANCEPRINCIPAL")
	log.InfoLog("Instance Principal flag", "ip", ip)
	if ip != "" {
		val, err := strconv.ParseBool(ip)
		if err != nil {
			configDetails.useInstancePrincipals = false
		}
		configDetails.useInstancePrincipals = val
	}

	vault := os.Getenv("VAULTDETAILS")
	log.InfoLog("Vault Details", "ocid", vault)
	if vault != "" {
		configDetails.vaultDetails = vault
	}

	SetUserConfigDetails(log)
	configDetails.auth.AuthType = resolveConfiguredAuthType(configDetails)

	return configDetails
}

func resolveConfiguredAuthType(config osokConfig) string {
	authCfg := config.auth
	if authType := authCfg.NormalizedAuthType(); authType != "" {
		return authType
	}
	if config.useInstancePrincipals && !authCfg.HasAnyUserPrincipalInput() {
		return AuthTypeInstancePrincipal
	}
	return authCfg.EffectiveAuthType()
}

func SetUserConfigDetails(log loggerutil.OSOKLogger) {
	log.InfoLog("Setting UserConfig Details")
	authType := os.Getenv("AUTH_TYPE")
	if authType != "" {
		configDetails.auth.AuthType = authType
	}

	user := os.Getenv("USER")
	if user != "" {
		configDetails.auth.User = user
	}

	tenancy := os.Getenv("TENANCY")
	if tenancy != "" {
		configDetails.auth.Tenancy = tenancy
	}

	region := os.Getenv("REGION")
	if region != "" {
		configDetails.auth.Region = region
	}

	fingerprint := os.Getenv("FINGERPRINT")
	if fingerprint != "" {
		configDetails.auth.Fingerprint = fingerprint
	}

	passphrase := os.Getenv("PASSPHRASE")
	if passphrase != "" {
		configDetails.auth.Passphrase = passphrase
	}

	privateKey := os.Getenv("PRIVATEKEY")
	if privateKey != "" {
		configDetails.auth.PrivateKey = privateKey
	}

	configFilePath := os.Getenv("OCI_CONFIG_FILE_PATH")
	if configFilePath != "" {
		configDetails.auth.ConfigFilePath = configFilePath
	}

	configFileProfile := os.Getenv("OCI_CONFIG_PROFILE")
	if configFileProfile != "" {
		configDetails.auth.ConfigFileProfile = configFileProfile
	}

	instancePrincipalDelegationToken := os.Getenv("INSTANCE_PRINCIPAL_DELEGATION_TOKEN")
	if instancePrincipalDelegationToken != "" {
		configDetails.auth.InstancePrincipalDelegationToken = instancePrincipalDelegationToken
	}

	resourcePrincipalDelegationToken := os.Getenv("RESOURCE_PRINCIPAL_DELEGATION_TOKEN")
	if resourcePrincipalDelegationToken != "" {
		configDetails.auth.ResourcePrincipalDelegationToken = resourcePrincipalDelegationToken
	}

	instancePrincipalLeafCertificatePath := os.Getenv("INSTANCE_PRINCIPAL_LEAF_CERTIFICATE_PATH")
	if instancePrincipalLeafCertificatePath != "" {
		configDetails.auth.InstancePrincipalLeafCertificatePath = instancePrincipalLeafCertificatePath
	}

	instancePrincipalLeafPrivateKeyPath := os.Getenv("INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PATH")
	if instancePrincipalLeafPrivateKeyPath != "" {
		configDetails.auth.InstancePrincipalLeafPrivateKeyPath = instancePrincipalLeafPrivateKeyPath
	}

	instancePrincipalLeafPrivateKeyPassphrase := os.Getenv("INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PASSPHRASE")
	if instancePrincipalLeafPrivateKeyPassphrase != "" {
		configDetails.auth.InstancePrincipalLeafPrivateKeyPassphrase = instancePrincipalLeafPrivateKeyPassphrase
	}

	instancePrincipalIntermediateCertificatePaths := os.Getenv("INSTANCE_PRINCIPAL_INTERMEDIATE_CERTIFICATE_PATHS")
	if instancePrincipalIntermediateCertificatePaths != "" {
		configDetails.auth.InstancePrincipalIntermediateCertPathList = instancePrincipalIntermediateCertificatePaths
	}
}

//func AddFlags() error {
//
// // We need to identify how to pass the UserAuthConfig
// flag.BoolVar(&configDetails.useInstancePrincipals, "useInstancePrincipals", configDetails.useInstancePrincipals, ""+
//    "Set to true if Instance Principals needs to be used for authentication")
// flag.StringVar(&configDetails.vaultDetails, "vaultDetails", configDetails.vaultDetails,
//    "OCI Vault details to be use for storing the access information is the service")
// return nil
//}
