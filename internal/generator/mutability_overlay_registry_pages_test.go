package generator

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func TestResolveMutabilityOverlayRegistryPagesForExplicitCoreService(t *testing.T) {
	t.Parallel()

	cfg := loadCheckedInConfig(t)
	services, err := cfg.SelectDefaultActiveOrExplicitServices("core", false)
	if err != nil {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(core) error = %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("SelectDefaultActiveOrExplicitServices(core) returned %d services, want 1", len(services))
	}
	if got := services[0].SelectedKinds(); !slices.Equal(got, []string{
		"Instance",
		"Drg",
		"InternetGateway",
		"NatGateway",
		"NetworkSecurityGroup",
		"RouteTable",
		"SecurityList",
		"ServiceGateway",
		"Subnet",
		"Vcn",
	}) {
		t.Fatalf("core SelectedKinds() = %v, want instance plus core-network split kinds", got)
	}

	contract, err := newMutabilityOverlayDocsContract("7.22.0")
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v", err)
	}

	packages := []*PackageModel{
		{
			Service: services[0],
			Resources: []ResourceModel{
				{Kind: "Instance", Formal: newFormalModel("core", "instance", "oci_core_instance")},
				{Kind: "Drg"},
				{Kind: "InternetGateway"},
				{Kind: "NatGateway"},
				{Kind: "NetworkSecurityGroup"},
				{Kind: "RouteTable"},
				{Kind: "SecurityList"},
				{Kind: "ServiceGateway"},
				{Kind: "Subnet"},
				{Kind: "Vcn"},
			},
		},
	}
	inventory := []formal.ProviderInventoryEntry{
		{Service: "core", Kind: "Drg", Slug: "drg", TerraformName: "oci_core_drg", RegistrationType: "resource"},
		{Service: "core", Kind: "InternetGateway", Slug: "internetgateway", TerraformName: "oci_core_internet_gateway", RegistrationType: "resource"},
		{Service: "core", Kind: "NatGateway", Slug: "natgateway", TerraformName: "oci_core_nat_gateway", RegistrationType: "resource"},
		{Service: "core", Kind: "NetworkSecurityGroup", Slug: "networksecuritygroup", TerraformName: "oci_core_network_security_group", RegistrationType: "resource"},
		{Service: "core", Kind: "RouteTable", Slug: "routetable", TerraformName: "oci_core_route_table", RegistrationType: "resource"},
		{Service: "core", Kind: "SecurityList", Slug: "securitylist", TerraformName: "oci_core_security_list", RegistrationType: "resource"},
		{Service: "core", Kind: "ServiceGateway", Slug: "servicegateway", TerraformName: "oci_core_service_gateway", RegistrationType: "resource"},
		{Service: "core", Kind: "Subnet", Slug: "subnet", TerraformName: "oci_core_subnet", RegistrationType: "resource"},
		{Service: "core", Kind: "Vcn", Slug: "vcn", TerraformName: "oci_core_vcn", RegistrationType: "resource"},
	}

	targets, err := resolveMutabilityOverlayRegistryPages(packages, contract, inventory)
	if err != nil {
		t.Fatalf("resolveMutabilityOverlayRegistryPages() error = %v", err)
	}
	if len(targets) != len(packages[0].Resources) {
		t.Fatalf("resolved %d targets, want %d", len(targets), len(packages[0].Resources))
	}

	assertRegistryPageTarget(t, targets, "core", "Instance", "instance", "oci_core_instance", "providers/oracle/oci/7.22.0/docs/resources/core_instance", "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/core_instance")
	assertRegistryPageTarget(t, targets, "core", "Subnet", "", "oci_core_subnet", "providers/oracle/oci/7.22.0/docs/resources/core_subnet", "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/core_subnet")
	assertRegistryPageTarget(t, targets, "core", "Vcn", "", "oci_core_vcn", "providers/oracle/oci/7.22.0/docs/resources/core_vcn", "https://registry.terraform.io/providers/oracle/oci/7.22.0/docs/resources/core_vcn")
}

func TestResolveMutabilityOverlayRegistryPagesReportsRenamedResource(t *testing.T) {
	t.Parallel()

	contract, err := newMutabilityOverlayDocsContract("7.22.0")
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v", err)
	}

	packages := []*PackageModel{
		{
			Service: ServiceConfig{Service: "core"},
			Resources: []ResourceModel{
				{Kind: "Instance", Formal: newFormalModel("core", "instance", "oci_core_instance")},
			},
		},
	}
	inventory := []formal.ProviderInventoryEntry{
		{Service: "core", Kind: "Instance", Slug: "instance", TerraformName: "oci_core_compute_instance", RegistrationType: "resource"},
	}

	_, err = resolveMutabilityOverlayRegistryPages(packages, contract, inventory)
	if err == nil {
		t.Fatal("resolveMutabilityOverlayRegistryPages() unexpectedly succeeded")
	}

	var mappingErr *mutabilityOverlayRegistryPageMappingError
	if !errors.As(err, &mappingErr) {
		t.Fatalf("resolveMutabilityOverlayRegistryPages() error = %v, want mutabilityOverlayRegistryPageMappingError", err)
	}
	if mappingErr.Reason != mutabilityOverlayRegistryPageErrorRenamedResource {
		t.Fatalf("mapping error reason = %q, want %q", mappingErr.Reason, mutabilityOverlayRegistryPageErrorRenamedResource)
	}
	if mappingErr.ProviderResource != "oci_core_instance" {
		t.Fatalf("mapping error providerResource = %q, want %q", mappingErr.ProviderResource, "oci_core_instance")
	}
	if !slices.Equal(mappingErr.Candidates, []string{"oci_core_compute_instance"}) {
		t.Fatalf("mapping error candidates = %v, want [oci_core_compute_instance]", mappingErr.Candidates)
	}
}

func TestResolveMutabilityOverlayRegistryPagesReportsAmbiguousProviderInventoryMatch(t *testing.T) {
	t.Parallel()

	contract, err := newMutabilityOverlayDocsContract("7.22.0")
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v", err)
	}

	packages := []*PackageModel{
		{
			Service: ServiceConfig{Service: "core"},
			Resources: []ResourceModel{
				{Kind: "Subnet"},
			},
		},
	}
	inventory := []formal.ProviderInventoryEntry{
		{Service: "core", Kind: "Subnet", Slug: "subnet", TerraformName: "oci_core_subnet", RegistrationType: "resource"},
		{Service: "core", Kind: "Subnet", Slug: "subnet", TerraformName: "oci_core_virtual_subnet", RegistrationType: "resource"},
	}

	_, err = resolveMutabilityOverlayRegistryPages(packages, contract, inventory)
	if err == nil {
		t.Fatal("resolveMutabilityOverlayRegistryPages() unexpectedly succeeded")
	}

	var mappingErr *mutabilityOverlayRegistryPageMappingError
	if !errors.As(err, &mappingErr) {
		t.Fatalf("resolveMutabilityOverlayRegistryPages() error = %v, want mutabilityOverlayRegistryPageMappingError", err)
	}
	if mappingErr.Reason != mutabilityOverlayRegistryPageErrorAmbiguousMatch {
		t.Fatalf("mapping error reason = %q, want %q", mappingErr.Reason, mutabilityOverlayRegistryPageErrorAmbiguousMatch)
	}
	if !slices.Equal(mappingErr.Candidates, []string{"oci_core_subnet", "oci_core_virtual_subnet"}) {
		t.Fatalf("mapping error candidates = %v, want [oci_core_subnet oci_core_virtual_subnet]", mappingErr.Candidates)
	}
}

func TestResolveMutabilityOverlayRegistryPagesReportsMissingTerraformName(t *testing.T) {
	t.Parallel()

	contract, err := newMutabilityOverlayDocsContract("7.22.0")
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v", err)
	}

	packages := []*PackageModel{
		{
			Service: ServiceConfig{Service: "queue"},
			Resources: []ResourceModel{
				{Kind: "Queue"},
			},
		},
	}

	_, err = resolveMutabilityOverlayRegistryPages(packages, contract, nil)
	if err == nil {
		t.Fatal("resolveMutabilityOverlayRegistryPages() unexpectedly succeeded")
	}

	var mappingErr *mutabilityOverlayRegistryPageMappingError
	if !errors.As(err, &mappingErr) {
		t.Fatalf("resolveMutabilityOverlayRegistryPages() error = %v, want mutabilityOverlayRegistryPageMappingError", err)
	}
	if mappingErr.Reason != mutabilityOverlayRegistryPageErrorMissingTerraformName {
		t.Fatalf("mapping error reason = %q, want %q", mappingErr.Reason, mutabilityOverlayRegistryPageErrorMissingTerraformName)
	}
	if mappingErr.Service != "queue" || mappingErr.Kind != "Queue" {
		t.Fatalf("mapping error target = %s/%s, want queue/Queue", mappingErr.Service, mappingErr.Kind)
	}
}

func TestResolveMutabilityOverlayRegistryPagesReportsURLDerivationFailure(t *testing.T) {
	t.Parallel()

	contract, err := newMutabilityOverlayDocsContract("7.22.0")
	if err != nil {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v", err)
	}

	packages := []*PackageModel{
		{
			Service: ServiceConfig{Service: "core"},
			Resources: []ResourceModel{
				{Kind: "Instance", Formal: newFormalModel("core", "instance", "core_instance")},
			},
		},
	}

	_, err = resolveMutabilityOverlayRegistryPages(packages, contract, nil)
	if err == nil {
		t.Fatal("resolveMutabilityOverlayRegistryPages() unexpectedly succeeded")
	}

	var mappingErr *mutabilityOverlayRegistryPageMappingError
	if !errors.As(err, &mappingErr) {
		t.Fatalf("resolveMutabilityOverlayRegistryPages() error = %v, want mutabilityOverlayRegistryPageMappingError", err)
	}
	if mappingErr.Reason != mutabilityOverlayRegistryPageErrorURLDerivationFailed {
		t.Fatalf("mapping error reason = %q, want %q", mappingErr.Reason, mutabilityOverlayRegistryPageErrorURLDerivationFailed)
	}
	if mappingErr.ProviderResource != "core_instance" {
		t.Fatalf("mapping error providerResource = %q, want %q", mappingErr.ProviderResource, "core_instance")
	}
}

func TestNewMutabilityOverlayDocsContractRejectsBlankVersion(t *testing.T) {
	t.Parallel()

	_, err := newMutabilityOverlayDocsContract("   ")
	if err == nil {
		t.Fatal("newMutabilityOverlayDocsContract() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "terraformDocsVersion") {
		t.Fatalf("newMutabilityOverlayDocsContract() error = %v, want terraformDocsVersion validation", err)
	}
}

func newFormalModel(service, slug, providerResource string) *FormalModel {
	return &FormalModel{
		Reference: FormalReferenceModel{
			Service: service,
			Slug:    slug,
		},
		Binding: formal.ControllerBinding{
			Import: formal.ImportModel{
				ProviderResource: providerResource,
			},
		},
	}
}

func assertRegistryPageTarget(
	t *testing.T,
	targets []mutabilityOverlayRegistryPageTarget,
	service string,
	kind string,
	formalSlug string,
	providerResource string,
	registryPath string,
	registryURL string,
) {
	t.Helper()

	target, ok := findRegistryPageTarget(targets, service, kind)
	if !ok {
		t.Fatalf("registry page target %s/%s not found in %#v", service, kind, targets)
	}
	if target.FormalSlug != formalSlug {
		t.Fatalf("%s/%s formalSlug = %q, want %q", service, kind, target.FormalSlug, formalSlug)
	}
	if target.ProviderResource != providerResource {
		t.Fatalf("%s/%s providerResource = %q, want %q", service, kind, target.ProviderResource, providerResource)
	}
	if target.RegistryPath != registryPath {
		t.Fatalf("%s/%s registryPath = %q, want %q", service, kind, target.RegistryPath, registryPath)
	}
	if target.RegistryURL != registryURL {
		t.Fatalf("%s/%s registryURL = %q, want %q", service, kind, target.RegistryURL, registryURL)
	}
}

func findRegistryPageTarget(
	targets []mutabilityOverlayRegistryPageTarget,
	service string,
	kind string,
) (mutabilityOverlayRegistryPageTarget, bool) {
	for _, target := range targets {
		if target.Service == service && target.Kind == kind {
			return target, true
		}
	}
	return mutabilityOverlayRegistryPageTarget{}, false
}
