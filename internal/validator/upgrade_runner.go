package validator

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/config"
	reportpkg "github.com/oracle/oci-service-operator/internal/validator/report"
	upgradepkg "github.com/oracle/oci-service-operator/internal/validator/upgrade"
)

func RunUpgrade(options config.Options) (upgradepkg.Report, string, error) {
	if err := options.ValidateUpgrade(); err != nil {
		return upgradepkg.Report{}, "", err
	}

	analyzer, err := upgradepkg.NewAnalyzer()
	if err != nil {
		return upgradepkg.Report{}, "", err
	}
	report, err := analyzer.Analyze(strings.TrimSpace(options.UpgradeFrom), strings.TrimSpace(options.UpgradeTo), options.ProviderPath)
	if err != nil {
		return upgradepkg.Report{}, "", err
	}
	if options.WantsUpgradeSelection() {
		selectedStructs, err := loadUpgradeSelectedStructs(options)
		if err != nil {
			return upgradepkg.Report{}, "", err
		}
		report = filterUpgradeReportByStructs(report, selectedStructs)
	} else if options.HasServiceFilter() {
		report = filterUpgradeReportByService(report, strings.ToLower(strings.TrimSpace(options.Service)))
	}
	formatValue := strings.TrimSpace(strings.ToLower(options.Format))
	if formatValue == "" {
		formatValue = string(reportpkg.FormatTable)
	}
	switch reportpkg.Format(formatValue) {
	case reportpkg.FormatTable, reportpkg.FormatMarkdown, reportpkg.FormatJSON:
		// ok
	default:
		return report, "", fmt.Errorf("unknown format %q", options.Format)
	}
	rendered, err := upgradepkg.Render(report, reportpkg.Format(formatValue))
	if err != nil {
		return report, "", err
	}
	return report, rendered, nil
}

func filterUpgradeReportByService(report upgradepkg.Report, service string) upgradepkg.Report {
	if service == "" {
		return report
	}
	prefix := service + "."
	filtered := report
	filtered.Structs = make([]upgradepkg.StructDiff, 0, len(report.Structs))
	for _, diff := range report.Structs {
		if strings.HasPrefix(diff.StructType, prefix) {
			filtered.Structs = append(filtered.Structs, diff)
		}
	}
	filtered.AllowlistSuggestions = make([]upgradepkg.AllowlistSuggestion, 0, len(report.AllowlistSuggestions))
	for _, suggestion := range report.AllowlistSuggestions {
		if strings.HasPrefix(upgradeSuggestionStructType(suggestion.Path), prefix) {
			filtered.AllowlistSuggestions = append(filtered.AllowlistSuggestions, suggestion)
		}
	}
	return filtered
}

func filterUpgradeReportByStructs(report upgradepkg.Report, allowed map[string]struct{}) upgradepkg.Report {
	filtered := report
	filtered.Structs = make([]upgradepkg.StructDiff, 0, len(report.Structs))
	for _, diff := range report.Structs {
		if _, ok := allowed[diff.StructType]; ok {
			filtered.Structs = append(filtered.Structs, diff)
		}
	}
	filtered.AllowlistSuggestions = make([]upgradepkg.AllowlistSuggestion, 0, len(report.AllowlistSuggestions))
	for _, suggestion := range report.AllowlistSuggestions {
		if _, ok := allowed[upgradeSuggestionStructType(suggestion.Path)]; ok {
			filtered.AllowlistSuggestions = append(filtered.AllowlistSuggestions, suggestion)
		}
	}
	return filtered
}

func upgradeSuggestionStructType(path string) string {
	structType, _, found := strings.Cut(path, ".fields.")
	if !found {
		return path
	}
	return structType
}
