/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/pkg/errors"

	. "github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
)

type AuthConfigProvider struct {
	Log       loggerutil.OSOKLogger
	Validator func(context.Context, common.ConfigurationProvider, OsokConfig) bool
}

func (configProvider *AuthConfigProvider) GetAuthProvider(osokConfig OsokConfig) (common.ConfigurationProvider, error) {
	var config common.ConfigurationProvider
	var err error
	if osokConfig != nil {
		configProvider.Log.InfoLog("The OSOK config is present, validating config parameters")
		authCfg := osokConfig.Auth()

		switch {
		case authCfg.WantsSecurityToken():
			configProvider.Log.InfoLog("Security token auth requested, validating configuration")
			config, err = configProvider.securityTokenProvider(authCfg)
			if err != nil {
				return nil, err
			}
			if !configProvider.validateProvider(context.Background(), config, osokConfig) {
				configProvider.Log.InfoLog("Security token configuration is not valid. Setup will now terminate")
				err = errors.New("Failed to instantiate Security Token auth provider")
			}
		case authCfg.HasAnyUserPrincipalField():
			if !authCfg.HasCompleteUserPrincipal() {
				err = errors.New("incomplete user principal configuration")
				configProvider.Log.ErrorLog(err, "User principals are configured incompletely")
				return nil, err
			}
			configProvider.Log.InfoLog("User principals available, validating user credentials")
			config = common.NewRawConfigurationProvider(
				authCfg.Tenancy,
				authCfg.User,
				authCfg.Region,
				authCfg.Fingerprint,
				authCfg.PrivateKey,
				common.String(authCfg.Passphrase))

			//If user principals failed to validate, setup will stop
			if !configProvider.validateProvider(context.Background(), config, osokConfig) {
				configProvider.Log.InfoLog("User Principals are not valid. Setup will now terminate")
				err = errors.New("Failed to instantiate User Principals")
			}
		default:
			configProvider.Log.InfoLog("User Principals are not present, switching to Instance principals")
			config, err = auth.InstancePrincipalConfigurationProvider()
			if err != nil {
				configProvider.Log.InfoLog("Failed to instantiate InstancePrincipals")
			}
		}
	} else {
		configProvider.Log.InfoLog("The OSOK config is not present. Using default Config provider")
		config = common.DefaultConfigProvider()
	}
	return config, err
}

func (configProvider *AuthConfigProvider) securityTokenProvider(authCfg UserAuthConfig) (common.ConfigurationProvider, error) {
	configFilePath := strings.TrimSpace(authCfg.ConfigFilePath)
	if configFilePath == "" {
		configFilePath = DefaultSecurityTokenConfigFilePath
	}

	profile := strings.TrimSpace(authCfg.ConfigFileProfile)
	if profile == "" {
		profile = DefaultSecurityTokenConfigProfile
	}

	provider, err := common.ConfigurationProviderForSessionTokenWithProfile(configFilePath, profile, authCfg.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("create security token configuration provider: %w", err)
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) validateProvider(ctx context.Context, provider common.ConfigurationProvider, config OsokConfig) bool {
	if configProvider.Validator != nil {
		return configProvider.Validator(ctx, provider, config)
	}
	return configProvider.authValidate(ctx, provider, config)
}

func (configProvider *AuthConfigProvider) authValidate(ctx context.Context, provider common.ConfigurationProvider, config OsokConfig) bool {
	configProvider.Log.InfoLog("Validating the Configuration Provider")
	tenancy, err := resolveValidationTenancy(provider, config)
	if err != nil {
		configProvider.Log.ErrorLog(err, "unable to determine tenancy for auth validation")
		return false
	}
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

func resolveValidationTenancy(provider common.ConfigurationProvider, config OsokConfig) (string, error) {
	if tenancy := strings.TrimSpace(config.Auth().Tenancy); tenancy != "" {
		return tenancy, nil
	}
	tenancy, err := provider.TenancyOCID()
	if err != nil {
		return "", fmt.Errorf("resolve tenancy from provider: %w", err)
	}
	return tenancy, nil
}
