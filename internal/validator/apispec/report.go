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
	b.WriteString("## API Coverage\n\n")
	if len(report.Structs) == 0 {
		b.WriteString("All tracked SDK fields are exposed by the OSOK API surfaces.\n")
		return b.String()
	}
	for _, strct := range report.Structs {
		targetLabel := fmt.Sprintf("`%s`", strct.SDKStruct)
		if strct.SDKStruct == "" {
			targetLabel = "`untracked`"
			if isIntentionalUntrackedReason(strct.TrackingReason) {
				targetLabel = "`intentionally untracked`"
			}
		}
		b.WriteString(fmt.Sprintf("### %s → %s\n\n", displayTargetName(strct), targetLabel))
		if strct.TrackingStatus == TrackingStatusUntracked {
			if strct.TrackingReason != "" {
				b.WriteString(strct.TrackingReason + "\n\n")
			}
			continue
		}
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
	b.WriteString("API coverage\n")
	if len(report.Structs) == 0 {
		b.WriteString("All tracked SDK fields are exposed by the OSOK API surfaces.\n")
		return b.String()
	}
	b.WriteString("\nAPI TARGET / SDK STRUCT                   PRESENT  MISSING  EXTRA\n")
	for _, strct := range report.Structs {
		sdkStruct := strct.SDKStruct
		if sdkStruct == "" {
			sdkStruct = "[untracked]"
			if isIntentionalUntrackedReason(strct.TrackingReason) {
				sdkStruct = "[intentionally untracked]"
			}
		}
		b.WriteString(fmt.Sprintf("%-35s %-40s %7d %8d %6d\n", displayTargetName(strct), sdkStruct, len(strct.PresentFields), len(strct.MissingFields), len(strct.ExtraSpecFields)))
		if strct.TrackingStatus == TrackingStatusUntracked {
			trackingLabel := strct.TrackingStatus
			if isIntentionalUntrackedReason(strct.TrackingReason) {
				trackingLabel = "intentionally untracked"
			}
			b.WriteString(fmt.Sprintf("  Tracking: %s", trackingLabel))
			if strct.TrackingReason != "" {
				b.WriteString(fmt.Sprintf(" (%s)", strct.TrackingReason))
			}
			b.WriteString("\n\n")
			continue
		}
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

func displayTargetName(strct StructReport) string {
	if strings.TrimSpace(strct.APISurface) == "" || strct.APISurface == apiSurfaceSpec {
		return strct.Spec
	}
	return fmt.Sprintf("%s (%s)", strct.Spec, strct.APISurface)
}
