package formal

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

// ProviderInventoryEntry identifies one formal scaffold row discoverable from the
// terraform-provider-oci registration surface.
type ProviderInventoryEntry struct {
	Service          string
	Slug             string
	Kind             string
	TerraformName    string
	RegistrationType string
}

// DiscoverProviderInventory loads terraform-provider-oci and returns the unique
// formal controller inventory discoverable from registered resources and
// data sources.
func DiscoverProviderInventory(providerPath string) ([]ProviderInventoryEntry, error) {
	providerRoot, err := resolveProviderRoot(strings.TrimSpace(providerPath))
	if err != nil {
		return nil, err
	}

	env := append(os.Environ(), "GOWORK=off")
	cfg := &packages.Config{
		Dir:  providerRoot,
		Env:  env,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax,
	}

	loaded, err := packages.Load(cfg, "./internal/service/...")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(loaded) > 0 {
		return nil, fmt.Errorf("failed to load provider packages from %s", providerRoot)
	}

	index := buildProviderIndex(loaded)
	resourceRegs := collectRegistrations(index, "RegisterResource")
	dataSourceRegs := collectRegistrations(index, "RegisterDatasource")

	entries := map[string]ProviderInventoryEntry{}
	for _, reg := range resourceRegs {
		entry, err := providerInventoryEntryFromResource(reg)
		if err != nil {
			return nil, err
		}
		if err := upsertProviderInventoryEntry(entries, entry); err != nil {
			return nil, err
		}
	}
	for _, reg := range dataSourceRegs {
		entry, err := providerInventoryEntryFromDatasource(reg)
		if err != nil {
			return nil, err
		}
		if err := upsertProviderInventoryEntry(entries, entry); err != nil {
			return nil, err
		}
	}

	out := make([]ProviderInventoryEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		if out[i].Slug != out[j].Slug {
			return out[i].Slug < out[j].Slug
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].RegistrationType != out[j].RegistrationType {
			return out[i].RegistrationType < out[j].RegistrationType
		}
		return out[i].TerraformName < out[j].TerraformName
	})
	return out, nil
}

func providerInventoryEntryFromResource(reg providerRegistration) (ProviderInventoryEntry, error) {
	service, err := providerServiceName(reg.Package.PkgPath)
	if err != nil {
		return ProviderInventoryEntry{}, err
	}
	kind := providerKindFromConstructor(reg.Constructor)
	if kind == "" {
		return ProviderInventoryEntry{}, fmt.Errorf("%s: resource constructor %q did not resolve a kind", reg.Package.PkgPath, reg.Constructor)
	}
	return ProviderInventoryEntry{
		Service:          service,
		Slug:             normalizeFormalSlug(kind),
		Kind:             kind,
		TerraformName:    reg.TerraformName,
		RegistrationType: "resource",
	}, nil
}

func providerInventoryEntryFromDatasource(reg providerRegistration) (ProviderInventoryEntry, error) {
	service, err := providerServiceName(reg.Package.PkgPath)
	if err != nil {
		return ProviderInventoryEntry{}, err
	}

	kind := providerKindFromConstructor(reg.Constructor)
	constructorFn := reg.Package.Funcs[reg.Constructor]
	if constructorFn == nil {
		return ProviderInventoryEntry{}, fmt.Errorf("%s: missing datasource constructor %s", reg.Package.PkgPath, reg.Constructor)
	}

	resourceLit, err := resourceLiteralFromFunc(constructorFn)
	if err == nil {
		_, resourceConstructor := findListCollection(extractSchemaMap(resourceLit))
		if resourceConstructor != "" {
			if linkedKind := providerKindFromConstructor(resourceConstructor); linkedKind != "" {
				kind = linkedKind
			}
		}
	}

	if kind == "" {
		return ProviderInventoryEntry{}, fmt.Errorf("%s: datasource constructor %q did not resolve a kind", reg.Package.PkgPath, reg.Constructor)
	}
	return ProviderInventoryEntry{
		Service:          service,
		Slug:             normalizeFormalSlug(kind),
		Kind:             kind,
		TerraformName:    reg.TerraformName,
		RegistrationType: "datasource",
	}, nil
}

func upsertProviderInventoryEntry(entries map[string]ProviderInventoryEntry, entry ProviderInventoryEntry) error {
	key := entry.Service + "/" + entry.Slug
	current, exists := entries[key]
	if !exists {
		entries[key] = entry
		return nil
	}
	if current.Kind != entry.Kind {
		if !strings.EqualFold(current.Kind, entry.Kind) {
			return fmt.Errorf("provider inventory key %s has conflicting kinds %q and %q", key, current.Kind, entry.Kind)
		}
		current.Kind = preferredProviderKind(current.Kind, entry.Kind)
	}
	if current.RegistrationType == "datasource" && entry.RegistrationType == "resource" {
		current.TerraformName = entry.TerraformName
		current.RegistrationType = entry.RegistrationType
	}
	entries[key] = current
	return nil
}

func preferredProviderKind(current, candidate string) string {
	if uppercaseRuneCount(candidate) > uppercaseRuneCount(current) {
		return candidate
	}
	if uppercaseRuneCount(candidate) == uppercaseRuneCount(current) && len(candidate) > len(current) {
		return candidate
	}
	return current
}

func uppercaseRuneCount(value string) int {
	count := 0
	for _, r := range value {
		if unicode.IsUpper(r) {
			count++
		}
	}
	return count
}

func providerServiceName(pkgPath string) (string, error) {
	const marker = "/internal/service/"
	index := strings.Index(pkgPath, marker)
	if index < 0 {
		return "", fmt.Errorf("provider package %q is not under internal/service", pkgPath)
	}

	rest := strings.TrimPrefix(pkgPath[index:], marker)
	segment := rest
	if slash := strings.Index(segment, "/"); slash >= 0 {
		segment = segment[:slash]
	}
	segment = normalizeFormalSlug(segment)
	if segment == "" {
		return "", fmt.Errorf("provider package %q did not resolve a service name", pkgPath)
	}
	return segment, nil
}

func providerKindFromConstructor(constructor string) string {
	constructor = strings.TrimSpace(constructor)
	for _, suffix := range []string{"Resource", "DataSource"} {
		if strings.HasSuffix(constructor, suffix) {
			return strings.TrimSuffix(constructor, suffix)
		}
	}
	return constructor
}

func normalizeFormalSlug(value string) string {
	var b strings.Builder
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
