package validator

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/validator/config"
	reportpkg "github.com/oracle/oci-service-operator/pkg/validator/report"
	upgradepkg "github.com/oracle/oci-service-operator/pkg/validator/upgrade"
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
