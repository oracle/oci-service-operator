/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/identity"
	mysqlsdk "github.com/oracle/oci-go-sdk/v65/mysql"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type dbSystemAvailabilityDomainClient interface {
	ListAvailabilityDomains(context.Context, identity.ListAvailabilityDomainsRequest) (identity.ListAvailabilityDomainsResponse, error)
	GetCompartment(context.Context, identity.GetCompartmentRequest) (identity.GetCompartmentResponse, error)
}

var newDbSystemAvailabilityDomainClient = func(provider common.ConfigurationProvider) (dbSystemAvailabilityDomainClient, error) {
	return identity.NewIdentityClientWithConfigurationProvider(provider)
}

func init() {
	registerDbSystemRuntimeHooksMutator(func(manager *DbSystemServiceManager, hooks *DbSystemRuntimeHooks) {
		applyDbSystemRuntimeHooks(manager, hooks)
	})
}

func applyDbSystemRuntimeHooks(manager *DbSystemServiceManager, hooks *DbSystemRuntimeHooks) {
	if manager == nil || hooks == nil {
		return
	}

	hooks.BuildCreateBody = func(ctx context.Context, resource *mysqlv1beta1.DbSystem, namespace string) (any, error) {
		return buildDbSystemCreateDetails(ctx, manager.Provider, manager.CredentialClient, resource, namespace)
	}
}

func buildDbSystemCreateDetails(
	ctx context.Context,
	provider common.ConfigurationProvider,
	credentialClient credhelper.CredentialClient,
	resource *mysqlv1beta1.DbSystem,
	namespace string,
) (mysqlsdk.CreateDbSystemDetails, error) {
	resolvedSpec, err := generatedruntime.ResolveSpecValue(resource, ctx, credentialClient, namespace)
	if err != nil {
		return mysqlsdk.CreateDbSystemDetails{}, err
	}

	payload, err := json.Marshal(resolvedSpec)
	if err != nil {
		return mysqlsdk.CreateDbSystemDetails{}, fmt.Errorf("marshal resolved mysql dbsystem spec: %w", err)
	}

	var details mysqlsdk.CreateDbSystemDetails
	if err := json.Unmarshal(payload, &details); err != nil {
		return mysqlsdk.CreateDbSystemDetails{}, fmt.Errorf("decode mysql create request body: %w", err)
	}

	normalizedAD, err := normalizeDbSystemAvailabilityDomain(ctx, provider, resource.Spec.CompartmentId, resource.Spec.AvailabilityDomain)
	if err != nil {
		return mysqlsdk.CreateDbSystemDetails{}, err
	}
	if normalizedAD == "" {
		details.AvailabilityDomain = nil
	} else {
		details.AvailabilityDomain = common.String(normalizedAD)
	}

	// Preserve standalone intent. The generic JSON projection drops explicit false
	// booleans behind json:",omitempty", but mysql create expects the same
	// always-projected value the legacy manager sent.
	details.IsHighlyAvailable = common.Bool(resource.Spec.IsHighlyAvailable)

	return details, nil
}

func normalizeDbSystemAvailabilityDomain(
	ctx context.Context,
	provider common.ConfigurationProvider,
	compartmentID string,
	requested string,
) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" || provider == nil {
		return requested, nil
	}

	identityClient, err := newDbSystemAvailabilityDomainClient(provider)
	if err != nil {
		return "", fmt.Errorf("initialize mysql dbsystem availabilityDomain client: %w", err)
	}

	tenancy, err := resolveDbSystemAvailabilityDomainTenancy(ctx, identityClient, compartmentID)
	if err != nil {
		return "", err
	}

	response, err := identityClient.ListAvailabilityDomains(ctx, identity.ListAvailabilityDomainsRequest{
		CompartmentId: common.String(tenancy),
	})
	if err != nil {
		return "", fmt.Errorf("list mysql dbsystem availability domains: %w", err)
	}

	available := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if item.Name == nil {
			continue
		}
		name := strings.TrimSpace(*item.Name)
		if name == "" {
			continue
		}
		available = append(available, name)
		if name == requested {
			return requested, nil
		}
	}

	requestedSuffix := availabilityDomainSuffix(requested)
	matches := make([]string, 0, len(available))
	for _, name := range available {
		if availabilityDomainSuffix(name) == requestedSuffix {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 1:
		// Preserve explicit caller-supplied aliases. MySQL accepts historical or
		// subnet-derived tenancy tokens such as qqZb:... even when Identity lists
		// the same regional AD under a different alias token for the current auth
		// context. Only expand suffix-only inputs.
		if strings.Contains(requested, ":") {
			return requested, nil
		}
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("mysql dbsystem availabilityDomain %q does not match the current auth context availability domains %q", requested, strings.Join(available, ", "))
	default:
		return "", fmt.Errorf("mysql dbsystem availabilityDomain %q maps to multiple current auth context availability domains %q", requested, strings.Join(matches, ", "))
	}
}

func resolveDbSystemAvailabilityDomainTenancy(
	ctx context.Context,
	identityClient dbSystemAvailabilityDomainClient,
	compartmentID string,
) (string, error) {
	current := strings.TrimSpace(compartmentID)
	if current == "" {
		return "", fmt.Errorf("resolve mysql dbsystem availabilityDomain tenancy: empty compartment id")
	}
	if strings.HasPrefix(current, "ocid1.tenancy.") {
		return current, nil
	}

	for depth := 0; depth < 16; depth++ {
		response, err := identityClient.GetCompartment(ctx, identity.GetCompartmentRequest{
			CompartmentId: common.String(current),
		})
		if err != nil {
			return "", fmt.Errorf("resolve mysql dbsystem availabilityDomain tenancy for compartment %q: %w", current, err)
		}

		parent := ""
		if response.Compartment.CompartmentId != nil {
			parent = strings.TrimSpace(*response.Compartment.CompartmentId)
		}
		if parent == "" {
			return "", fmt.Errorf("resolve mysql dbsystem availabilityDomain tenancy for compartment %q: empty parent compartment", current)
		}
		if strings.HasPrefix(parent, "ocid1.tenancy.") {
			return parent, nil
		}
		if parent == current {
			return "", fmt.Errorf("resolve mysql dbsystem availabilityDomain tenancy for compartment %q: parent points to self", current)
		}
		current = parent
	}

	return "", fmt.Errorf("resolve mysql dbsystem availabilityDomain tenancy for compartment %q: exceeded parent traversal limit", compartmentID)
}

func availabilityDomainSuffix(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.Index(name, ":"); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}
