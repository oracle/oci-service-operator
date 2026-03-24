package formalscaffold

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/formal"
	"github.com/oracle/oci-service-operator/internal/generator"
)

type CoverageReport struct {
	Root          string
	ConfigPath    string
	ExpectedRows  int
	ManifestRows  int
	MissingRows   int
	ProviderKinds int
}

func (r CoverageReport) String() string {
	var b strings.Builder
	b.WriteString("formal scaffold coverage passed\n")
	fmt.Fprintf(&b, "- root: %s\n", filepath.ToSlash(r.Root))
	fmt.Fprintf(&b, "- config: %s\n", filepath.ToSlash(r.ConfigPath))
	fmt.Fprintf(&b, "- provider kinds discovered: %d\n", r.ProviderKinds)
	fmt.Fprintf(&b, "- expected rows: %d\n", r.ExpectedRows)
	fmt.Fprintf(&b, "- manifest rows: %d\n", r.ManifestRows)
	return b.String()
}

func VerifyCoverage(opts Options) (CoverageReport, error) {
	report := CoverageReport{}
	if strings.TrimSpace(opts.ProviderPath) == "" {
		return report, fmt.Errorf("provider source path must not be empty")
	}
	if strings.TrimSpace(opts.Root) == "" {
		return report, fmt.Errorf("formal root must not be empty")
	}
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return report, fmt.Errorf("generator config path must not be empty")
	}

	formalRoot, err := filepath.Abs(strings.TrimSpace(opts.Root))
	if err != nil {
		return report, err
	}
	configPath, err := filepath.Abs(strings.TrimSpace(opts.ConfigPath))
	if err != nil {
		return report, err
	}
	report.Root = formalRoot
	report.ConfigPath = configPath

	if err := requireDirectory(formalRoot); err != nil {
		return report, err
	}

	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return report, err
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(configPath), "..", "..", ".."))

	publishedEntries, err := discoverPublishedKinds(repoRoot, cfg)
	if err != nil {
		return report, err
	}
	providerEntries, err := discoverProviderKinds(strings.TrimSpace(opts.ProviderPath))
	if err != nil {
		return report, err
	}
	expectedEntries, err := mergeInventoryEntries(publishedEntries, providerEntries)
	if err != nil {
		return report, err
	}
	report.ExpectedRows = len(expectedEntries)
	report.ProviderKinds = len(providerEntries)

	catalog, err := formal.LoadCatalog(formalRoot)
	if err != nil {
		return report, err
	}
	report.ManifestRows = len(catalog.Controllers)

	manifestKinds := map[string]string{}
	for _, binding := range catalog.Controllers {
		manifestKinds[rowKey(binding.Manifest.Service, binding.Manifest.Slug)] = binding.Manifest.Kind
	}

	var problems []string
	for _, entry := range expectedEntries {
		key := rowKey(entry.Service, entry.Slug)
		kind, ok := manifestKinds[key]
		if !ok {
			report.MissingRows++
			problems = append(problems, fmt.Sprintf("missing formal scaffold row for %s/%s (%s)", entry.Service, entry.Slug, entry.Kind))
			continue
		}
		if kind != entry.Kind {
			problems = append(problems, fmt.Sprintf("formal scaffold row %s/%s has kind %q, want %q", entry.Service, entry.Slug, kind, entry.Kind))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return report, fmt.Errorf("formal scaffold coverage failed:\n- %s", strings.Join(problems, "\n- "))
	}
	return report, nil
}
