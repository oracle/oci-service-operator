/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"os"
	"strconv"
)

var (
	configDetails osokConfig
)

func GetConfigDetails(log loggerutil.OSOKLogger) osokConfig {
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

	return configDetails
}

func SetUserConfigDetails(log loggerutil.OSOKLogger) {
	log.InfoLog("Setting UserConfig Details")
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
