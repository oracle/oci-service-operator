package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/allowlist"
	"github.com/oracle/oci-service-operator/internal/validator/apispec"
	"github.com/oracle/oci-service-operator/internal/validator/config"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
	"github.com/oracle/oci-service-operator/internal/validator/provider"
	reportpkg "github.com/oracle/oci-service-operator/internal/validator/report"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
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

	apiReport, err := apispec.BuildReport(sdkStructs, allow)
	if err != nil {
		return diff.Report{}, "", err
	}

	if options.HasServiceFilter() {
		service := strings.ToLower(strings.TrimSpace(options.Service))
		result = filterControllerReportByService(result, service)
		apiReport = filterAPIReportByService(apiReport, service)
	}

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

	rendered, err := render(result, apiReport, options)
	if err != nil {
		return diff.Report{}, "", err
	}

	if options.FailOnNew && reportpkg.HasNewActionable(result) {
		return result, rendered, ErrNewActionable
	}

	return result, rendered, nil
}

func filterControllerReportByService(report diff.Report, service string) diff.Report {
	if service == "" {
		return report
	}
	out := diff.Report{Structs: make([]diff.StructReport, 0, len(report.Structs))}
	prefix := service + "."
	for _, structReport := range report.Structs {
		if strings.HasPrefix(structReport.StructType, prefix) {
			out.Structs = append(out.Structs, structReport)
		}
	}
	return out
}

func filterAPIReportByService(report apispec.Report, service string) apispec.Report {
	if service == "" {
		return report
	}
	out := apispec.Report{Structs: make([]apispec.StructReport, 0, len(report.Structs))}
	prefix := service + "."
	for _, structReport := range report.Structs {
		if strings.HasPrefix(structReport.SDKStruct, prefix) {
			out.Structs = append(out.Structs, structReport)
		}
	}
	return out
}

func render(controller diff.Report, api apispec.Report, options config.Options) (string, error) {
	format := strings.TrimSpace(strings.ToLower(options.Format))
	if format == "" {
		format = string(reportpkg.FormatTable)
	}
	switch reportpkg.Format(format) {
	case reportpkg.FormatTable, reportpkg.FormatMarkdown:
		controllerText, err := reportpkg.Render(controller, reportpkg.Format(format))
		if err != nil {
			return "", err
		}
		apiText, err := apispec.Render(api, reportpkg.Format(format))
		if err != nil {
			return "", err
		}
		controllerText = strings.TrimRight(controllerText, "\n")
		apiText = strings.TrimRight(apiText, "\n")
		switch {
		case controllerText == "" && apiText == "":
			return "\n", nil
		case controllerText == "":
			return apiText + "\n", nil
		case apiText == "":
			return controllerText + "\n", nil
		default:
			return controllerText + "\n\n" + apiText + "\n", nil
		}
	case reportpkg.FormatJSON:
		controllerJSON, err := reportpkg.Render(controller, reportpkg.FormatJSON)
		if err != nil {
			return "", err
		}
		apiJSON, err := apispec.Render(api, reportpkg.FormatJSON)
		if err != nil {
			return "", err
		}
		combined := map[string]json.RawMessage{
			"controller": json.RawMessage(strings.TrimSpace(controllerJSON)),
			"api":        json.RawMessage(strings.TrimSpace(apiJSON)),
		}
		buf, err := json.MarshalIndent(combined, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf) + "\n", nil
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
