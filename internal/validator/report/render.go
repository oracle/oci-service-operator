package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/diff"
)

type Format string

const (
	FormatTable    Format = "table"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

func Render(report diff.Report, format Format) (string, error) {
	switch format {
	case FormatTable:
		return renderTable(report), nil
	case FormatMarkdown:
		return renderMarkdown(report), nil
	case FormatJSON:
		buf := &bytes.Buffer{}
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return "", err
		}
		return buf.String(), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

func renderTable(report diff.Report) string {
	var b strings.Builder
	for _, structReport := range report.Structs {
		fmt.Fprintf(&b, "%s (%.1f%% coverage)\n", structReport.StructType, structReport.Coverage.Percent)
		for _, field := range structReport.Fields {
			fmt.Fprintf(&b, "  - %s: %s", field.FieldName, field.Status)
			switch field.Status {
			case diff.FieldStatusUsed:
				fmt.Fprintf(&b, " (%d refs)", len(field.References))
			case diff.FieldStatusDeprecated, diff.FieldStatusReadOnly, diff.FieldStatusIntentionallyOmitted, diff.FieldStatusFutureConsideration, diff.FieldStatusPotentialGap, diff.FieldStatusUnclassified:
				if field.Reason != "" {
					fmt.Fprintf(&b, " (%s)", field.Reason)
				}
			}
			if field.PreviousStatus != "" && field.PreviousStatus != field.Status {
				fmt.Fprintf(&b, " [previous: %s]", field.PreviousStatus)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderMarkdown(report diff.Report) string {
	var b strings.Builder
	for _, structReport := range report.Structs {
		fmt.Fprintf(&b, "### %s (%.1f%% coverage)\n\n", structReport.StructType, structReport.Coverage.Percent)
		for _, field := range structReport.Fields {
			fmt.Fprintf(&b, "- **%s** — %s", field.FieldName, field.Status)
			switch field.Status {
			case diff.FieldStatusUsed:
				fmt.Fprintf(&b, " (%d refs)", len(field.References))
			case diff.FieldStatusDeprecated, diff.FieldStatusReadOnly, diff.FieldStatusIntentionallyOmitted, diff.FieldStatusFutureConsideration, diff.FieldStatusPotentialGap, diff.FieldStatusUnclassified:
				if field.Reason != "" {
					fmt.Fprintf(&b, " (%s)", field.Reason)
				}
			}
			if field.PreviousStatus != "" && field.PreviousStatus != field.Status {
				fmt.Fprintf(&b, " _(previously %s)_", field.PreviousStatus)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func WithBaseline(current diff.Report, baseline diff.Report) diff.Report {
	baselineIndex := map[string]diff.StructReport{}
	for _, structReport := range baseline.Structs {
		baselineIndex[structReport.StructType] = structReport
	}

	out := diff.Report{Structs: make([]diff.StructReport, 0, len(current.Structs))}
	for _, structReport := range current.Structs {
		base := baselineIndex[structReport.StructType]
		fields := make([]diff.FieldReport, 0, len(structReport.Fields))
		baselineMap := map[string]diff.FieldReport{}
		for _, f := range base.Fields {
			baselineMap[f.FieldName] = f
		}
		for _, field := range structReport.Fields {
			if previous, ok := baselineMap[field.FieldName]; ok {
				field.PreviousStatus = previous.Status
			}
			fields = append(fields, field)
		}
		structReport.Fields = fields
		out.Structs = append(out.Structs, structReport)
	}

	sort.Slice(out.Structs, func(i, j int) bool { return out.Structs[i].StructType < out.Structs[j].StructType })
	return out
}

func HasNewActionable(report diff.Report) bool {
	for _, structReport := range report.Structs {
		for _, field := range structReport.Fields {
			if field.Status == diff.FieldStatusUnclassified && field.PreviousStatus != diff.FieldStatusUnclassified {
				return true
			}
			if field.Status == diff.FieldStatusPotentialGap && field.PreviousStatus != diff.FieldStatusPotentialGap {
				return true
			}
		}
	}
	return false
}
