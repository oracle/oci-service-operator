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

	configpkg "github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

type AuthConfigProvider struct {
	Log       loggerutil.OSOKLogger
	Validator func(context.Context, common.ConfigurationProvider, configpkg.OsokConfig) bool
}

func (configProvider *AuthConfigProvider) GetAuthProvider(osokConfig configpkg.OsokConfig) (common.ConfigurationProvider, error) {
	if osokConfig == nil {
		configProvider.Log.InfoLog("The OSOK config is not present. Using default Config provider")
		return common.DefaultConfigProvider(), nil
	} else {
		configProvider.Log.InfoLog("The OSOK config is present, validating config parameters")
	}

	authCfg := osokConfig.Auth()
	if authCfg.WantsSecurityToken() {
		return configProvider.securityTokenAuthProvider(osokConfig, authCfg)
	}
	if authCfg.HasAnyUserPrincipalField() {
		return configProvider.userPrincipalAuthProvider(osokConfig, authCfg)
	}
	return configProvider.instancePrincipalAuthProvider()
}

func (configProvider *AuthConfigProvider) securityTokenAuthProvider(osokConfig configpkg.OsokConfig, authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Security token auth requested, validating configuration")
	provider, err := configProvider.securityTokenProvider(authCfg)
	if err != nil {
		return nil, err
	}
	if !configProvider.validateProvider(context.Background(), provider, osokConfig) {
		configProvider.Log.InfoLog("Security token configuration is not valid. Setup will now terminate")
		return provider, errors.New("Failed to instantiate Security Token auth provider")
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) userPrincipalAuthProvider(osokConfig configpkg.OsokConfig, authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	if !authCfg.HasCompleteUserPrincipal() {
		err := errors.New("incomplete user principal configuration")
		configProvider.Log.ErrorLog(err, "User principals are configured incompletely")
		return nil, err
	}

	configProvider.Log.InfoLog("User principals available, validating user credentials")
	provider := common.NewRawConfigurationProvider(
		authCfg.Tenancy,
		authCfg.User,
		authCfg.Region,
		authCfg.Fingerprint,
		authCfg.PrivateKey,
		common.String(authCfg.Passphrase))

	if !configProvider.validateProvider(context.Background(), provider, osokConfig) {
		configProvider.Log.InfoLog("User Principals are not valid. Setup will now terminate")
		return provider, errors.New("Failed to instantiate User Principals")
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) instancePrincipalAuthProvider() (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("User Principals are not present, switching to Instance principals")
	provider, err := auth.InstancePrincipalConfigurationProvider()
	if err != nil {
		configProvider.Log.InfoLog("Failed to instantiate InstancePrincipals")
	}
	return provider, err
}

func (configProvider *AuthConfigProvider) securityTokenProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configFilePath := strings.TrimSpace(authCfg.ConfigFilePath)
	if configFilePath == "" {
		configFilePath = configpkg.DefaultSecurityTokenConfigFilePath
	}

	profile := strings.TrimSpace(authCfg.ConfigFileProfile)
	if profile == "" {
		profile = configpkg.DefaultSecurityTokenConfigProfile
	}

	provider, err := common.ConfigurationProviderForSessionTokenWithProfile(configFilePath, profile, authCfg.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("create security token configuration provider: %w", err)
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) validateProvider(ctx context.Context, provider common.ConfigurationProvider, config configpkg.OsokConfig) bool {
	if configProvider.Validator != nil {
		return configProvider.Validator(ctx, provider, config)
	}
	return configProvider.authValidate(ctx, provider, config)
}

func (configProvider *AuthConfigProvider) authValidate(ctx context.Context, provider common.ConfigurationProvider, config configpkg.OsokConfig) bool {
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
		name := ""
		if r.Items[0].Name != nil {
			name = *r.Items[0].Name
		}
		configProvider.Log.InfoLog("The ADs obtained during validation", "ADs", name)
	} else {
		configProvider.Log.InfoLog("No ADs obtained during validation")
	}
	return true
}

func resolveValidationTenancy(provider common.ConfigurationProvider, config configpkg.OsokConfig) (string, error) {
	if tenancy := strings.TrimSpace(config.Auth().Tenancy); tenancy != "" {
		return tenancy, nil
	}
	tenancy, err := provider.TenancyOCID()
	if err != nil {
		return "", fmt.Errorf("resolve tenancy from provider: %w", err)
	}
	return tenancy, nil
}
