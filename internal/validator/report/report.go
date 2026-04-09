package report

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/diff"
)

func RenderText(result diff.Report) string {
	if len(result.Structs) == 0 {
		return "no tracked SDK structs\n"
	}
	var builder strings.Builder
	for _, strct := range result.Structs {
		fmt.Fprintf(&builder, "%s (%.1f%% coverage)\n", strct.StructType, strct.Coverage.Percent)
		for _, field := range strct.Fields {
			if field.Status == diff.FieldStatusUsed {
				fmt.Fprintf(&builder, "  - %s: used (%d refs)\n", field.FieldName, len(field.References))
				continue
			}
			fmt.Fprintf(&builder, "  - %s: %s", field.FieldName, field.Status)
			if field.Reason != "" {
				fmt.Fprintf(&builder, " (%s)", field.Reason)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
