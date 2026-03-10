package diff

import (
	"sort"
	"strconv"

	"github.com/oracle/oci-service-operator/pkg/validator/allowlist"
	"github.com/oracle/oci-service-operator/pkg/validator/provider"
	"github.com/oracle/oci-service-operator/pkg/validator/sdk"
)

func BuildReport(sdkStructs []sdk.SDKStruct, providerAnalysis provider.Analysis, allow allowlist.Allowlist) Report {
	usageIndex := buildUsageIndex(providerAnalysis.Usages)

	structReports := make([]StructReport, 0, len(sdkStructs))
	for _, sdkStruct := range sdkStructs {
		allowFields := allow.Structs[sdkStruct.QualifiedName].Fields
		fieldReports := make([]FieldReport, 0, len(sdkStruct.Fields))
		eligible := 0
		used := 0

		for _, sdkField := range sdkStruct.Fields {
			references := append([]string(nil), usageIndex[sdkStruct.QualifiedName][sdkField.Name]...)
			allowField := allowFields[sdkField.Name]

			report := FieldReport{
				StructType:    sdkStruct.QualifiedName,
				FieldName:     sdkField.Name,
				FieldType:     sdkField.Type,
				JSONName:      sdkField.JSONName,
				Mandatory:     sdkField.Mandatory,
				Used:          len(references) > 0,
				Deprecated:    sdkField.Deprecated,
				ReadOnly:      sdkField.ReadOnly,
				Documentation: sdkField.Documentation,
				Status:        FieldStatusUnclassified,
				References:    references,
			}

			switch {
			case len(references) > 0:
				report.Status = FieldStatusUsed
			case sdkField.Deprecated:
				report.Status = FieldStatusDeprecated
				if report.Reason == "" {
					report.Reason = "Deprecated in the OCI SDK."
				}
			case sdkField.ReadOnly:
				report.Status = FieldStatusReadOnly
				if report.Reason == "" {
					report.Reason = "Read-only or server-managed in the OCI SDK."
				}
			case allowField.Status == allowlist.StatusIntentionallyOmitted:
				report.Status = FieldStatusIntentionallyOmitted
				report.Reason = allowField.Reason
			case allowField.Status == allowlist.StatusPotentialGap:
				report.Status = FieldStatusPotentialGap
				report.Reason = allowField.Reason
			case allowField.Status == allowlist.StatusFutureConsideration:
				report.Status = FieldStatusFutureConsideration
				report.Reason = allowField.Reason
			case allowField.Status == allowlist.StatusUsed:
				report.Status = FieldStatusUnclassified
				if report.Reason == "" {
					report.Reason = "Allowlist marks this field as used, but provider analysis found no usage."
				}
			default:
				if report.Reason == "" {
					report.Reason = "Field has no allowlist classification and no provider usage."
				}
			}

			if len(report.References) == 0 && len(allowField.References) > 0 {
				report.References = append([]string(nil), allowField.References...)
			}

			if isCoverageEligible(report.Status) {
				eligible++
				if report.Status == FieldStatusUsed {
					used++
				}
			}

			fieldReports = append(fieldReports, report)
		}

		sort.Slice(fieldReports, func(i, j int) bool {
			return fieldReports[i].FieldName < fieldReports[j].FieldName
		})

		percent := 100.0
		if eligible > 0 {
			percent = float64(used) / float64(eligible) * 100
		}

		structReports = append(structReports, StructReport{
			StructType: sdkStruct.QualifiedName,
			Coverage: Coverage{
				EligibleFields: eligible,
				UsedFields:     used,
				Percent:        percent,
			},
			Fields: fieldReports,
		})
	}

	sort.Slice(structReports, func(i, j int) bool {
		return structReports[i].StructType < structReports[j].StructType
	})

	return Report{Structs: structReports}
}

func buildUsageIndex(usages []provider.FieldUsage) map[string]map[string][]string {
	index := map[string]map[string][]string{}
	for _, usage := range usages {
		structFields := index[usage.StructType]
		if structFields == nil {
			structFields = map[string][]string{}
			index[usage.StructType] = structFields
		}
		reference := usage.File + ":" + strconv.Itoa(usage.Line)
		if !contains(structFields[usage.FieldName], reference) {
			structFields[usage.FieldName] = append(structFields[usage.FieldName], reference)
		}
		for field := range structFields {
			sort.Strings(structFields[field])
		}
	}
	return index
}

func isCoverageEligible(status FieldStatus) bool {
	return status != FieldStatusIntentionallyOmitted && status != FieldStatusDeprecated && status != FieldStatusReadOnly
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
