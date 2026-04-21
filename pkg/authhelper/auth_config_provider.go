/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/identity"

	configpkg "github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

type authProviderFactory func(configpkg.UserAuthConfig) (common.ConfigurationProvider, error)

type AuthConfigProvider struct {
	Log loggerutil.OSOKLogger

	Validator func(context.Context, common.ConfigurationProvider, configpkg.OsokConfig) bool

	SecurityTokenProviderFactory                    authProviderFactory
	UserPrincipalRawProviderFactory                 authProviderFactory
	UserPrincipalFileProviderFactory                authProviderFactory
	InstancePrincipalProviderFactory                authProviderFactory
	InstancePrincipalWithCertsProviderFactory       authProviderFactory
	ResourcePrincipalProviderFactory                authProviderFactory
	OKEWorkloadIdentityProviderFactory              authProviderFactory
	InstancePrincipalDelegationTokenProviderFactory authProviderFactory
	ResourcePrincipalDelegationTokenProviderFactory authProviderFactory
}

func (configProvider *AuthConfigProvider) GetAuthProvider(osokConfig configpkg.OsokConfig) (common.ConfigurationProvider, error) {
	if osokConfig == nil {
		configProvider.Log.InfoLog("The OSOK config is not present. Using default Config provider")
		return common.DefaultConfigProvider(), nil
	}

	configProvider.Log.InfoLog("The OSOK config is present, validating config parameters")
	authCfg := osokConfig.Auth()
	authType := authCfg.EffectiveAuthType()

	switch authType {
	case configpkg.AuthTypeSecurityToken:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.SecurityTokenProviderFactory,
			configProvider.securityTokenProvider,
		))
	case configpkg.AuthTypeUserPrincipal:
		return configProvider.userPrincipalAuthProvider(osokConfig, authCfg)
	case configpkg.AuthTypeInstancePrincipal:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.InstancePrincipalProviderFactory,
			configProvider.instancePrincipalProvider,
		))
	case configpkg.AuthTypeInstancePrincipalWithCerts:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.InstancePrincipalWithCertsProviderFactory,
			configProvider.instancePrincipalWithCertsProvider,
		))
	case configpkg.AuthTypeResourcePrincipal:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.ResourcePrincipalProviderFactory,
			configProvider.resourcePrincipalProvider,
		))
	case configpkg.AuthTypeOKEWorkloadIdentity:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.OKEWorkloadIdentityProviderFactory,
			configProvider.okeWorkloadIdentityProvider,
		))
	case configpkg.AuthTypeInstancePrincipalDelegationToken:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.InstancePrincipalDelegationTokenProviderFactory,
			configProvider.instancePrincipalDelegationTokenProvider,
		))
	case configpkg.AuthTypeResourcePrincipalDelegationToken:
		return configProvider.newValidatedProvider(authType, osokConfig, authCfg, providerFactoryOr(
			configProvider.ResourcePrincipalDelegationTokenProviderFactory,
			configProvider.resourcePrincipalDelegationTokenProvider,
		))
	case configpkg.AuthTypeWorkloadIdentityFederation, configpkg.AuthTypeOAuthDelegationToken:
		return nil, fmt.Errorf("auth type %q is not supported by the OCI Go SDK version pinned in this checkout", authType)
	default:
		return nil, fmt.Errorf("unsupported auth type %q", authType)
	}
}

func providerFactoryOr(factory authProviderFactory, fallback authProviderFactory) authProviderFactory {
	if factory != nil {
		return factory
	}
	return fallback
}

func (configProvider *AuthConfigProvider) newValidatedProvider(
	authType string,
	osokConfig configpkg.OsokConfig,
	authCfg configpkg.UserAuthConfig,
	factory authProviderFactory,
) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Resolving auth provider", "authType", authType)

	provider, err := factory(authCfg)
	if err != nil {
		return nil, err
	}

	if !configProvider.shouldValidate(authType) {
		configProvider.Log.InfoLog("Skipping auth validation for auth type", "authType", authType)
		return provider, nil
	}

	if !configProvider.validateProvider(context.Background(), provider, osokConfig) {
		return provider, fmt.Errorf("failed to validate auth type %q", authType)
	}

	return provider, nil
}

func (configProvider *AuthConfigProvider) userPrincipalAuthProvider(
	osokConfig configpkg.OsokConfig,
	authCfg configpkg.UserAuthConfig,
) (common.ConfigurationProvider, error) {
	switch {
	case authCfg.HasAnyUserPrincipalField():
		if !authCfg.HasCompleteUserPrincipal() {
			err := fmt.Errorf("incomplete user principal configuration")
			configProvider.Log.ErrorLog(err, "User principal auth is configured incompletely")
			return nil, err
		}
		return configProvider.newValidatedProvider(configpkg.AuthTypeUserPrincipal, osokConfig, authCfg, providerFactoryOr(
			configProvider.UserPrincipalRawProviderFactory,
			configProvider.userPrincipalRawProvider,
		))
	case authCfg.HasConfigFileReference():
		return configProvider.newValidatedProvider(configpkg.AuthTypeUserPrincipal, osokConfig, authCfg, providerFactoryOr(
			configProvider.UserPrincipalFileProviderFactory,
			configProvider.userPrincipalFileProvider,
		))
	default:
		err := fmt.Errorf("user principal auth requires raw OCI user fields or OCI config file inputs")
		configProvider.Log.ErrorLog(err, "User principal auth is configured incompletely")
		return nil, err
	}
}

func (configProvider *AuthConfigProvider) userPrincipalRawProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Creating raw user principal configuration provider")
	return common.NewRawConfigurationProvider(
		authCfg.Tenancy,
		authCfg.User,
		authCfg.Region,
		authCfg.Fingerprint,
		authCfg.PrivateKey,
		common.String(authCfg.Passphrase),
	), nil
}

func (configProvider *AuthConfigProvider) userPrincipalFileProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Creating file-backed user principal configuration provider")

	provider, err := common.ConfigurationProviderFromFileWithProfile(
		authCfg.EffectiveConfigFilePath(),
		authCfg.EffectiveConfigFileProfile(),
		authCfg.Passphrase,
	)
	if err != nil {
		return nil, fmt.Errorf("create file-backed user principal configuration provider: %w", err)
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) instancePrincipalProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	region := strings.TrimSpace(authCfg.Region)
	if region == "" {
		configProvider.Log.InfoLog("Creating instance principal configuration provider")
		return auth.InstancePrincipalConfigurationProvider()
	}

	configProvider.Log.InfoLog("Creating regional instance principal configuration provider", "region", region)
	return auth.InstancePrincipalConfigurationProviderForRegion(common.Region(region))
}

func (configProvider *AuthConfigProvider) instancePrincipalWithCertsProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	region := strings.TrimSpace(authCfg.Region)
	if region == "" {
		return nil, fmt.Errorf("instance principal with certs requires REGION")
	}

	leafCertificatePath := strings.TrimSpace(authCfg.InstancePrincipalLeafCertificatePath)
	if leafCertificatePath == "" {
		return nil, fmt.Errorf("instance principal with certs requires INSTANCE_PRINCIPAL_LEAF_CERTIFICATE_PATH")
	}

	leafPrivateKeyPath := strings.TrimSpace(authCfg.InstancePrincipalLeafPrivateKeyPath)
	if leafPrivateKeyPath == "" {
		return nil, fmt.Errorf("instance principal with certs requires INSTANCE_PRINCIPAL_LEAF_PRIVATE_KEY_PATH")
	}

	leafCertificate, err := os.ReadFile(leafCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("read instance principal leaf certificate %q: %w", leafCertificatePath, err)
	}

	leafPrivateKey, err := os.ReadFile(leafPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read instance principal leaf private key %q: %w", leafPrivateKeyPath, err)
	}

	intermediateCertificates := make([][]byte, 0, len(authCfg.IntermediateCertificatePaths()))
	for _, intermediatePath := range authCfg.IntermediateCertificatePaths() {
		intermediate, err := os.ReadFile(intermediatePath)
		if err != nil {
			return nil, fmt.Errorf("read instance principal intermediate certificate %q: %w", intermediatePath, err)
		}
		intermediateCertificates = append(intermediateCertificates, intermediate)
	}

	configProvider.Log.InfoLog("Creating instance principal with certs configuration provider", "region", region)
	provider, err := auth.InstancePrincipalConfigurationWithCerts(
		common.Region(region),
		leafCertificate,
		[]byte(authCfg.InstancePrincipalLeafPrivateKeyPassphrase),
		leafPrivateKey,
		intermediateCertificates,
	)
	if err != nil {
		return nil, fmt.Errorf("create instance principal with certs configuration provider: %w", err)
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) resourcePrincipalProvider(configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Creating resource principal configuration provider")
	return auth.ResourcePrincipalConfigurationProvider()
}

func (configProvider *AuthConfigProvider) okeWorkloadIdentityProvider(configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Creating OKE workload identity configuration provider")
	return auth.OkeWorkloadIdentityConfigurationProvider()
}

func (configProvider *AuthConfigProvider) instancePrincipalDelegationTokenProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	token := strings.TrimSpace(authCfg.InstancePrincipalDelegationToken)
	if token == "" {
		return nil, fmt.Errorf("instance principal delegation token auth requires INSTANCE_PRINCIPAL_DELEGATION_TOKEN")
	}

	region := strings.TrimSpace(authCfg.Region)
	if region == "" {
		configProvider.Log.InfoLog("Creating instance principal delegation token configuration provider")
		return auth.InstancePrincipalDelegationTokenConfigurationProvider(&token)
	}

	configProvider.Log.InfoLog("Creating regional instance principal delegation token configuration provider", "region", region)
	return auth.InstancePrincipalDelegationTokenConfigurationProviderForRegion(&token, common.Region(region))
}

func (configProvider *AuthConfigProvider) resourcePrincipalDelegationTokenProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	token := strings.TrimSpace(authCfg.ResourcePrincipalDelegationToken)
	if token == "" {
		return nil, fmt.Errorf("resource principal delegation token auth requires RESOURCE_PRINCIPAL_DELEGATION_TOKEN")
	}

	region := strings.TrimSpace(authCfg.Region)
	if region == "" {
		configProvider.Log.InfoLog("Creating resource principal delegation token configuration provider")
		return auth.ResourcePrincipalDelegationTokenConfigurationProvider(&token)
	}

	configProvider.Log.InfoLog("Creating regional resource principal delegation token configuration provider", "region", region)
	return auth.ResourcePrincipalDelegationTokenConfigurationProviderForRegion(&token, common.Region(region))
}

func (configProvider *AuthConfigProvider) securityTokenProvider(authCfg configpkg.UserAuthConfig) (common.ConfigurationProvider, error) {
	configProvider.Log.InfoLog("Creating security token configuration provider")

	provider, err := common.ConfigurationProviderForSessionTokenWithProfile(
		authCfg.EffectiveConfigFilePath(),
		authCfg.EffectiveConfigFileProfile(),
		authCfg.Passphrase,
	)
	if err != nil {
		return nil, fmt.Errorf("create security token configuration provider: %w", err)
	}
	return provider, nil
}

func (configProvider *AuthConfigProvider) shouldValidate(authType string) bool {
	switch authType {
	case configpkg.AuthTypeResourcePrincipal,
		configpkg.AuthTypeOKEWorkloadIdentity,
		configpkg.AuthTypeInstancePrincipalDelegationToken,
		configpkg.AuthTypeResourcePrincipalDelegationToken:
		return false
	default:
		return true
	}
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
