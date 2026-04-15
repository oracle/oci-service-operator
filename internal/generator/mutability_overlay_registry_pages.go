package generator

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/oracle/oci-service-operator/internal/formal"
)

const (
	mutabilityOverlayTerraformRegistryHost      = "registry.terraform.io"
	mutabilityOverlayTerraformRegistryNamespace = "oracle"
	mutabilityOverlayTerraformRegistryProvider  = "oci"
	mutabilityOverlayProviderResourcePrefix     = "oci_"

	mutabilityOverlayRegistryPageErrorMissingTerraformName = "missingTerraformName"
	mutabilityOverlayRegistryPageErrorAmbiguousMatch       = "ambiguousMatch"
	mutabilityOverlayRegistryPageErrorRenamedResource      = "renamedResource"
	mutabilityOverlayRegistryPageErrorURLDerivationFailed  = "registryURLDerivationFailed"
)

// mutabilityOverlayDocsContract captures the pinned Terraform Registry surface
// needed to turn provider resource names into exact versioned docs pages.
type mutabilityOverlayDocsContract struct {
	TerraformDocsVersion string
	RegistryHost         string
	ProviderNamespace    string
	ProviderName         string
}

// mutabilityOverlayRegistryPageTarget identifies one selected OSOK resource and
// the exact versioned Terraform Registry resource page that should be used as
// downstream docs input.
type mutabilityOverlayRegistryPageTarget struct {
	Service                string
	Kind                   string
	FormalSlug             string
	ProviderResource       string
	ProviderSourcePath     string
	ProviderSourceRevision string
	TerraformDocsVersion   string
	RegistryPath           string
	RegistryURL            string
}

// mutabilityOverlayRegistryPageMappingError captures an explicit discovery
// failure for one selected service/kind pair.
type mutabilityOverlayRegistryPageMappingError struct {
	Reason           string
	Service          string
	Kind             string
	FormalSlug       string
	ProviderResource string
	Candidates       []string
	Detail           string
}

func (e *mutabilityOverlayRegistryPageMappingError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "mutability overlay registry page mapping failed for service %q kind %q", e.Service, e.Kind)
	if strings.TrimSpace(e.FormalSlug) != "" {
		fmt.Fprintf(&b, " formalSpec %q", e.FormalSlug)
	}
	fmt.Fprintf(&b, ": %s", e.Reason)
	if strings.TrimSpace(e.ProviderResource) != "" {
		fmt.Fprintf(&b, " providerResource=%q", e.ProviderResource)
	}
	if len(e.Candidates) != 0 {
		fmt.Fprintf(&b, " candidates=%v", e.Candidates)
	}
	if strings.TrimSpace(e.Detail) != "" {
		fmt.Fprintf(&b, " (%s)", e.Detail)
	}
	return b.String()
}

func newMutabilityOverlayDocsContract(terraformDocsVersion string) (mutabilityOverlayDocsContract, error) {
	contract := mutabilityOverlayDocsContract{
		TerraformDocsVersion: strings.TrimSpace(terraformDocsVersion),
		RegistryHost:         mutabilityOverlayTerraformRegistryHost,
		ProviderNamespace:    mutabilityOverlayTerraformRegistryNamespace,
		ProviderName:         mutabilityOverlayTerraformRegistryProvider,
	}
	if err := contract.Validate(); err != nil {
		return mutabilityOverlayDocsContract{}, err
	}
	return contract, nil
}

func (c mutabilityOverlayDocsContract) Validate() error {
	var errs []string
	errs = append(errs, validateNonEmptyString("terraformDocsVersion", c.TerraformDocsVersion)...)
	errs = append(errs, validateNonEmptyString("registryHost", c.RegistryHost)...)
	errs = append(errs, validateNonEmptyString("providerNamespace", c.ProviderNamespace)...)
	errs = append(errs, validateNonEmptyString("providerName", c.ProviderName)...)
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

// resolveMutabilityOverlayRegistryPages maps the already-selected generator
// package resources to exact versioned Terraform Registry resource pages.
// Selection boundaries are inherited from the packages passed in, so the helper
// never widens the scrape surface beyond the current generator target set.
func resolveMutabilityOverlayRegistryPages(
	packages []*PackageModel,
	contract mutabilityOverlayDocsContract,
	providerInventory []formal.ProviderInventoryEntry,
) ([]mutabilityOverlayRegistryPageTarget, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}

	index := indexMutabilityOverlayProviderInventory(providerInventory)
	targets := make([]mutabilityOverlayRegistryPageTarget, 0)
	errs := make([]error, 0)
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		for _, resource := range pkg.Resources {
			target, err := resolveMutabilityOverlayRegistryPageTarget(pkg.Service.Service, resource, contract, index)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			targets = append(targets, target)
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Service != targets[j].Service {
			return targets[i].Service < targets[j].Service
		}
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		if targets[i].ProviderResource != targets[j].ProviderResource {
			return targets[i].ProviderResource < targets[j].ProviderResource
		}
		return targets[i].RegistryURL < targets[j].RegistryURL
	})
	if len(errs) != 0 {
		return targets, errors.Join(errs...)
	}
	return targets, nil
}

func resolveMutabilityOverlayRegistryPageTarget(
	service string,
	resource ResourceModel,
	contract mutabilityOverlayDocsContract,
	providerInventory map[string][]formal.ProviderInventoryEntry,
) (mutabilityOverlayRegistryPageTarget, error) {
	providerResource, formalSlug, err := resolveMutabilityOverlayProviderResource(service, resource, providerInventory)
	if err != nil {
		return mutabilityOverlayRegistryPageTarget{}, err
	}

	registryPath, registryURL, err := contract.versionedResourcePage(providerResource)
	if err != nil {
		return mutabilityOverlayRegistryPageTarget{}, &mutabilityOverlayRegistryPageMappingError{
			Reason:           mutabilityOverlayRegistryPageErrorURLDerivationFailed,
			Service:          service,
			Kind:             resource.Kind,
			FormalSlug:       formalSlug,
			ProviderResource: providerResource,
			Detail:           err.Error(),
		}
	}

	return mutabilityOverlayRegistryPageTarget{
		Service:              service,
		Kind:                 resource.Kind,
		FormalSlug:           formalSlug,
		ProviderResource:     providerResource,
		TerraformDocsVersion: contract.TerraformDocsVersion,
		RegistryPath:         registryPath,
		RegistryURL:          registryURL,
	}, nil
}

func resolveMutabilityOverlayProviderResource(
	service string,
	resource ResourceModel,
	providerInventory map[string][]formal.ProviderInventoryEntry,
) (string, string, error) {
	formalSlug := ""
	formalProviderResource := ""
	if resource.Formal != nil {
		formalSlug = strings.TrimSpace(resource.Formal.Reference.Slug)
		formalProviderResource = strings.TrimSpace(resource.Formal.Binding.Import.ProviderResource)
	}

	inventoryMatches := lookupMutabilityOverlayProviderInventory(providerInventory, service, resource.Kind, formalSlug)
	inventoryNames := uniqueSortedTerraformNames(inventoryMatches)
	switch {
	case formalProviderResource != "" && len(inventoryNames) == 0:
		return formalProviderResource, formalSlug, nil
	case formalProviderResource != "" && len(inventoryNames) == 1 && inventoryNames[0] == formalProviderResource:
		return formalProviderResource, formalSlug, nil
	case formalProviderResource != "" && len(inventoryNames) > 0 && slicesContainString(inventoryNames, formalProviderResource):
		return formalProviderResource, formalSlug, nil
	case formalProviderResource != "" && len(inventoryNames) > 0:
		return "", formalSlug, &mutabilityOverlayRegistryPageMappingError{
			Reason:           mutabilityOverlayRegistryPageErrorRenamedResource,
			Service:          service,
			Kind:             resource.Kind,
			FormalSlug:       formalSlug,
			ProviderResource: formalProviderResource,
			Candidates:       inventoryNames,
			Detail:           "formal binding providerResource does not match provider inventory",
		}
	case len(inventoryNames) == 1:
		return inventoryNames[0], formalSlug, nil
	case len(inventoryNames) > 1:
		return "", formalSlug, &mutabilityOverlayRegistryPageMappingError{
			Reason:     mutabilityOverlayRegistryPageErrorAmbiguousMatch,
			Service:    service,
			Kind:       resource.Kind,
			FormalSlug: formalSlug,
			Candidates: inventoryNames,
			Detail:     "multiple provider inventory matches resolved for the selected service and kind",
		}
	default:
		return "", formalSlug, &mutabilityOverlayRegistryPageMappingError{
			Reason:     mutabilityOverlayRegistryPageErrorMissingTerraformName,
			Service:    service,
			Kind:       resource.Kind,
			FormalSlug: formalSlug,
			Detail:     "no formal binding or provider inventory Terraform resource name was available",
		}
	}
}

func (c mutabilityOverlayDocsContract) versionedResourcePage(providerResource string) (string, string, error) {
	pageName, err := mutabilityOverlayRegistryPageName(providerResource)
	if err != nil {
		return "", "", err
	}

	registryPath := fmt.Sprintf(
		"providers/%s/%s/%s/docs/resources/%s",
		c.ProviderNamespace,
		c.ProviderName,
		c.TerraformDocsVersion,
		pageName,
	)
	registryURL := fmt.Sprintf("https://%s/%s", c.RegistryHost, registryPath)
	return registryPath, registryURL, nil
}

func mutabilityOverlayRegistryPageName(providerResource string) (string, error) {
	providerResource = strings.TrimSpace(providerResource)
	if !strings.HasPrefix(providerResource, mutabilityOverlayProviderResourcePrefix) {
		return "", fmt.Errorf("provider resource %q must start with %q", providerResource, mutabilityOverlayProviderResourcePrefix)
	}

	pageName := strings.TrimPrefix(providerResource, mutabilityOverlayProviderResourcePrefix)
	if strings.TrimSpace(pageName) == "" {
		return "", fmt.Errorf("provider resource %q does not include a docs page name after %q", providerResource, mutabilityOverlayProviderResourcePrefix)
	}
	return pageName, nil
}

func indexMutabilityOverlayProviderInventory(entries []formal.ProviderInventoryEntry) map[string][]formal.ProviderInventoryEntry {
	index := make(map[string][]formal.ProviderInventoryEntry)
	for _, entry := range entries {
		if !mutabilityOverlayInventoryEntrySupportsResourceDocs(entry) {
			continue
		}
		service := normalizeMutabilityOverlayIdentifier(entry.Service)
		if service == "" {
			continue
		}
		if kind := normalizeMutabilityOverlayIdentifier(entry.Kind); kind != "" {
			key := service + "\x00" + kind
			index[key] = appendUniqueProviderInventoryEntries(index[key], entry)
		}
		if slug := normalizeMutabilityOverlayIdentifier(entry.Slug); slug != "" {
			key := service + "\x00" + slug
			index[key] = appendUniqueProviderInventoryEntries(index[key], entry)
		}
	}
	return index
}

func lookupMutabilityOverlayProviderInventory(
	index map[string][]formal.ProviderInventoryEntry,
	service string,
	kind string,
	formalSlug string,
) []formal.ProviderInventoryEntry {
	serviceKey := normalizeMutabilityOverlayIdentifier(service)
	matches := append([]formal.ProviderInventoryEntry(nil), index[serviceKey+"\x00"+normalizeMutabilityOverlayIdentifier(kind)]...)
	if slugKey := normalizeMutabilityOverlayIdentifier(formalSlug); slugKey != "" {
		for _, entry := range index[serviceKey+"\x00"+slugKey] {
			matches = appendUniqueProviderInventoryEntries(matches, entry)
		}
	}
	return matches
}

func mutabilityOverlayInventoryEntrySupportsResourceDocs(entry formal.ProviderInventoryEntry) bool {
	switch strings.TrimSpace(entry.RegistrationType) {
	case "", "resource":
		return true
	default:
		return false
	}
}

func appendUniqueProviderInventoryEntries(
	existing []formal.ProviderInventoryEntry,
	entry formal.ProviderInventoryEntry,
) []formal.ProviderInventoryEntry {
	for _, current := range existing {
		if current.Service == entry.Service &&
			current.Slug == entry.Slug &&
			current.Kind == entry.Kind &&
			current.TerraformName == entry.TerraformName &&
			current.RegistrationType == entry.RegistrationType {
			return existing
		}
	}
	return append(existing, entry)
}

func uniqueSortedTerraformNames(entries []formal.ProviderInventoryEntry) []string {
	seen := make(map[string]struct{}, len(entries))
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.TerraformName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeMutabilityOverlayIdentifier(value string) string {
	var b strings.Builder
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func slicesContainString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
