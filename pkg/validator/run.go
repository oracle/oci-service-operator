package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/validator/allowlist"
	"github.com/oracle/oci-service-operator/pkg/validator/config"
	"github.com/oracle/oci-service-operator/pkg/validator/diff"
	"github.com/oracle/oci-service-operator/pkg/validator/provider"
	reportpkg "github.com/oracle/oci-service-operator/pkg/validator/report"
	"github.com/oracle/oci-service-operator/pkg/validator/sdk"
)

var ErrNewActionable = errors.New("new actionable gaps detected")

func Run(options config.Options) (diff.Report, string, error) {
	if err := options.Validate(); err != nil {
		return diff.Report{}, "", err
	}

	providerAnalyzer := provider.NewAnalyzer(options.ProviderPath)
	providerAnalysis, err := providerAnalyzer.Analyze()
	if err != nil {
		return diff.Report{}, "", err
	}

	var allow allowlist.Allowlist
	if options.HasAllowlist() {
		if _, err := os.Stat(options.AllowlistPath); err == nil {
			allow, err = allowlist.Load(options.AllowlistPath)
			if err != nil {
				return diff.Report{}, "", err
			}
		}
	}

	sdkAnalyzer, err := sdk.NewAnalyzer(options.ProviderPath)
	if err != nil {
		return diff.Report{}, "", err
	}
	sdkStructs, err := sdkAnalyzer.AnalyzeAll()
	if err != nil {
		return diff.Report{}, "", err
	}

	result := diff.BuildReport(sdkStructs, providerAnalysis, allow)

	if options.WantsBaselineWrite() {
		if err := writeBaseline(options.WriteBaseline, result); err != nil {
			return diff.Report{}, "", err
		}
	}

	if options.HasBaseline() {
		baseline, err := loadBaseline(options.BaselinePath)
		if err != nil {
			return diff.Report{}, "", err
		}
		result = reportpkg.WithBaseline(result, baseline)
	}

	rendered, err := render(result, options)
	if err != nil {
		return diff.Report{}, "", err
	}

	if options.FailOnNew && reportpkg.HasNewActionable(result) {
		return result, rendered, ErrNewActionable
	}

	return result, rendered, nil
}

func render(rep diff.Report, options config.Options) (string, error) {
	format := strings.TrimSpace(strings.ToLower(options.Format))
	if format == "" {
		format = string(reportpkg.FormatTable)
	}
	switch reportpkg.Format(format) {
	case reportpkg.FormatTable, reportpkg.FormatMarkdown, reportpkg.FormatJSON:
		return reportpkg.Render(rep, reportpkg.Format(format))
	default:
		return "", fmt.Errorf("unknown format %q", options.Format)
	}
}

func writeBaseline(path string, report diff.Report) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("baseline path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func loadBaseline(path string) (diff.Report, error) {
	var report diff.Report
	contents, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(contents, &report); err != nil {
		return diff.Report{}, err
	}
	return report, nil
}
