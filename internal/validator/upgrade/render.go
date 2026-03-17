package upgrade

import (
	"encoding/json"
	"fmt"
	"strings"

	reportpkg "github.com/oracle/oci-service-operator/internal/validator/report"
)

func Render(report Report, format reportpkg.Format) (string, error) {
	switch format {
	case reportpkg.FormatJSON:
		return renderJSON(report)
	case reportpkg.FormatMarkdown:
		return renderMarkdown(report), nil
	case reportpkg.FormatTable, "":
		return renderTable(report), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func renderJSON(report Report) (string, error) {
	if report.Structs == nil {
		report.Structs = []StructDiff{}
	}
	if report.AllowlistSuggestions == nil {
		report.AllowlistSuggestions = []AllowlistSuggestion{}
	}
	contents, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(contents) + "\n", nil
}

func renderMarkdown(report Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# OCI SDK Upgrade Report\n\nFrom `%s` to `%s`.\n\n", report.FromVersion, report.ToVersion))
	if report.ComparedToOSOK {
		b.WriteString("Current OSOK controller usage is included in the diff details.\n\n")
	}
	if len(report.Structs) == 0 {
		b.WriteString("No struct-level differences detected.\n")
		return b.String()
	}
	b.WriteString("## Struct Changes\n\n")
	for _, diff := range report.Structs {
		b.WriteString(fmt.Sprintf("### `%s`\n\n", diff.StructType))
		if len(diff.AddedFields) > 0 {
			b.WriteString("**Added fields**\n\n")
			for _, field := range diff.AddedFields {
				b.WriteString(fmt.Sprintf("- `%s` (json=`%s`, mandatory=%t, deprecated=%t, readOnly=%t", field.Name, field.JSONName, field.Mandatory, field.Deprecated, field.ReadOnly))
				if report.ComparedToOSOK {
					b.WriteString(fmt.Sprintf(", usedByOSOK=%t", field.UsedByOSOK))
				}
				b.WriteString(")\n")
				if len(field.References) > 0 {
					b.WriteString(fmt.Sprintf("  - references: `%s`\n", strings.Join(field.References, "`, `")))
				}
			}
			b.WriteString("\n")
		}
		if len(diff.RemovedFields) > 0 {
			b.WriteString("**Removed fields**\n\n")
			for _, field := range diff.RemovedFields {
				b.WriteString(fmt.Sprintf("- `%s`", field.Name))
				if report.ComparedToOSOK {
					b.WriteString(fmt.Sprintf(" (usedByOSOK=%t)", field.UsedByOSOK))
				}
				b.WriteString("\n")
				if len(field.References) > 0 {
					b.WriteString(fmt.Sprintf("  - references: `%s`\n", strings.Join(field.References, "`, `")))
				}
			}
			b.WriteString("\n")
		}
		if len(diff.ChangedFields) > 0 {
			b.WriteString("**Changed field metadata**\n\n")
			for _, field := range diff.ChangedFields {
				b.WriteString(fmt.Sprintf("- `%s`: json `%s` → `%s`, mandatory `%t` → `%t`, deprecated `%t` → `%t`, readOnly `%t` → `%t`",
					field.FieldName,
					field.From.JSONName, field.To.JSONName,
					field.From.Mandatory, field.To.Mandatory,
					field.From.Deprecated, field.To.Deprecated,
					field.From.ReadOnly, field.To.ReadOnly,
				))
				if report.ComparedToOSOK {
					b.WriteString(fmt.Sprintf(", usedByOSOK=%t", field.UsedByOSOK))
				}
				b.WriteString("\n")
				if len(field.References) > 0 {
					b.WriteString(fmt.Sprintf("  - references: `%s`\n", strings.Join(field.References, "`, `")))
				}
			}
			b.WriteString("\n")
		}
	}
	if len(report.AllowlistSuggestions) > 0 {
		b.WriteString("## Draft Allowlist Suggestions\n\n```yaml\n")
		for _, suggestion := range report.AllowlistSuggestions {
			b.WriteString(fmt.Sprintf("- path: %s\n  status: %s\n  reason: %s\n", suggestion.Path, suggestion.Status, suggestion.Reason))
		}
		b.WriteString("```\n")
	}
	return b.String()
}

func renderTable(report Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("upgrade %s -> %s\n", report.FromVersion, report.ToVersion))
	if report.ComparedToOSOK {
		b.WriteString("compared against current OSOK field usage\n")
	}
	if len(report.Structs) == 0 {
		b.WriteString("no struct differences detected\n")
		return b.String()
	}
	b.WriteString("\nSTRUCT                              ADDED  REMOVED  CHANGED  OSOK_USED\n")
	for _, diff := range report.Structs {
		b.WriteString(fmt.Sprintf("%-35s %5d %8d %8d %9d\n",
			diff.StructType,
			len(diff.AddedFields),
			len(diff.RemovedFields),
			len(diff.ChangedFields),
			countStructUsage(diff),
		))
	}
	if len(report.AllowlistSuggestions) > 0 {
		b.WriteString("\nALLOWLIST SUGGESTIONS\n")
		for _, suggestion := range report.AllowlistSuggestions {
			b.WriteString(fmt.Sprintf("- %s [%s]: %s\n", suggestion.Path, suggestion.Status, suggestion.Reason))
		}
	}
	return b.String()
}

func countStructUsage(diff StructDiff) int {
	count := 0
	for _, field := range diff.AddedFields {
		if field.UsedByOSOK {
			count++
		}
	}
	for _, field := range diff.RemovedFields {
		if field.UsedByOSOK {
			count++
		}
	}
	for _, field := range diff.ChangedFields {
		if field.UsedByOSOK {
			count++
		}
	}
	return count
}
