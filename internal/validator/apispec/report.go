package apispec

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
	case reportpkg.FormatTable:
		return renderTable(report), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func renderJSON(report Report) (string, error) {
	if report.Structs == nil {
		report.Structs = []StructReport{}
	}
	contents, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(contents) + "\n", nil
}

func renderMarkdown(report Report) string {
	var b strings.Builder
	b.WriteString("## API Spec Coverage\n\n")
	if len(report.Structs) == 0 {
		b.WriteString("All tracked SDK fields are exposed by the OSOK API specs.\n")
		return b.String()
	}
	for _, strct := range report.Structs {
		b.WriteString(fmt.Sprintf("### %s → `%s`\n\n", strct.Spec, strct.SDKStruct))
		if len(strct.PresentFields) > 0 {
			b.WriteString("**Present fields**\n\n")
			for _, field := range strct.PresentFields {
				b.WriteString(fmt.Sprintf("- **%s** — present", field.FieldName))
				if field.Mandatory {
					b.WriteString(" (mandatory)")
				}
				if field.Reason != "" {
					b.WriteString(fmt.Sprintf(" (%s)", field.Reason))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
		if len(strct.MissingFields) > 0 {
			b.WriteString("**Missing fields**\n\n")
			for _, field := range strct.MissingFields {
				b.WriteString(fmt.Sprintf("- **%s** — status: %s", field.FieldName, field.Status))
				if field.Mandatory {
					b.WriteString(", mandatory")
				}
				if field.Reason != "" {
					b.WriteString(fmt.Sprintf(" (%s)", field.Reason))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		} else {
			b.WriteString("**Missing fields**\n\n- None.\n\n")
		}
		if len(strct.ExtraSpecFields) > 0 {
			b.WriteString("**API-only fields**\n\n")
			for _, field := range strct.ExtraSpecFields {
				b.WriteString(fmt.Sprintf("- **%s** — status: %s", field.FieldName, field.Status))
				if field.Reason != "" {
					b.WriteString(fmt.Sprintf(" (%s)", field.Reason))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		} else {
			b.WriteString("**API-only fields**\n\n- None.\n\n")
		}
	}
	return b.String()
}

func renderTable(report Report) string {
	var b strings.Builder
	b.WriteString("API spec coverage\n")
	if len(report.Structs) == 0 {
		b.WriteString("All tracked SDK fields are exposed by the OSOK API specs.\n")
		return b.String()
	}
	b.WriteString("\nSPEC / SDK STRUCT                         PRESENT  MISSING  EXTRA\n")
	for _, strct := range report.Structs {
		b.WriteString(fmt.Sprintf("%-35s %-40s %7d %8d %6d\n", strct.Spec, strct.SDKStruct, len(strct.PresentFields), len(strct.MissingFields), len(strct.ExtraSpecFields)))
		if len(strct.PresentFields) > 0 {
			b.WriteString("  Present:\n")
			for _, field := range strct.PresentFields {
				b.WriteString(fmt.Sprintf("    + %s (mandatory=%t)\n", field.FieldName, field.Mandatory))
			}
		}
		if len(strct.MissingFields) > 0 {
			b.WriteString("  Missing:\n")
			for _, field := range strct.MissingFields {
				b.WriteString(fmt.Sprintf("    - %s (mandatory=%t, status=%s", field.FieldName, field.Mandatory, field.Status))
				if field.Reason != "" {
					b.WriteString(fmt.Sprintf(", %s", field.Reason))
				}
				b.WriteString(")\n")
			}
		}
		if len(strct.ExtraSpecFields) > 0 {
			b.WriteString("  API-only:\n")
			for _, field := range strct.ExtraSpecFields {
				b.WriteString(fmt.Sprintf("    * %s (status=%s", field.FieldName, field.Status))
				if field.Reason != "" {
					b.WriteString(fmt.Sprintf(", %s", field.Reason))
				}
				b.WriteString(")\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
