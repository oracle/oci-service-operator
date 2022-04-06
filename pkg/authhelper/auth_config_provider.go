/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import (
	"context"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/pkg/errors"

	. "github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
)

type AuthConfigProvider struct {
	Log loggerutil.OSOKLogger
}

func (configProvider *AuthConfigProvider) GetAuthProvider(osokConfig OsokConfig) (common.ConfigurationProvider, error) {
	var config common.ConfigurationProvider
	var err error
	if osokConfig != nil {
		configProvider.Log.InfoLog("The OSOK config is present, validating config parameters")

		//Check if user principals are present
		if reflect.DeepEqual(osokConfig.Auth(), UserAuthConfig{}) {
			configProvider.Log.InfoLog("User Principals are not present, switching to Instance principals")
			config, err = auth.InstancePrincipalConfigurationProvider()
			if err != nil {
				configProvider.Log.InfoLog("Failed to instantiate InstancePrincipals")
			}
		} else {
			configProvider.Log.InfoLog("User principals available, validating user credentials")
			config = common.NewRawConfigurationProvider(
				osokConfig.Auth().Tenancy,
				osokConfig.Auth().User,
				osokConfig.Auth().Region,
				osokConfig.Auth().Fingerprint,
				osokConfig.Auth().PrivateKey,
				common.String(osokConfig.Auth().Passphrase))

			//If user principals failed to validate, setup will stop
			if !configProvider.authValidate(context.Background(), config, osokConfig) {
				configProvider.Log.InfoLog("User Principals are not valid. Setup will now terminate")
				err = errors.New("Failed to instantiate User Principals")
			}
		}
	} else {
		configProvider.Log.InfoLog("The OSOK config is not present. Using default Config provider")
		config = common.DefaultConfigProvider()
	}
	return config, err
}

func (configProvider *AuthConfigProvider) authValidate(ctx context.Context, provider common.ConfigurationProvider, config OsokConfig) bool {
	configProvider.Log.InfoLog("Validating the Configuration Provider")
	tenancy := config.Auth().Tenancy
	// Validating the provider to list the ADs
	request := identity.ListAvailabilityDomainsRequest{
		CompartmentId: &tenancy,
	}
	identClient, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		configProvider.Log.ErrorLog(err, "unable to validate and instantiate using the auth provider.")
		return false
	}
	r, err := identClient.ListAvailabilityDomains(ctx, request)
	if err != nil {
		configProvider.Log.ErrorLog(err, "unable to validate the authentication provider.")
		return false
	}
	if len(r.Items) > 0 {
		configProvider.Log.InfoLog("The ADs obtained during validation", "ADs", conversions.DeRefString(r.Items[0].Name))
	} else {
		configProvider.Log.InfoLog("No ADs obtained during validation")
	}
	return true
}
