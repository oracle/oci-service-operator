package apispec

import (
	"reflect"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/pkg/validator/allowlist"
	"github.com/oracle/oci-service-operator/pkg/validator/diff"
	"github.com/oracle/oci-service-operator/pkg/validator/sdk"
)

type Report struct {
	Structs []StructReport `json:"structs"`
}

type StructReport struct {
	Spec            string        `json:"spec"`
	SDKStruct       string        `json:"sdkStruct"`
	PresentFields   []FieldReport `json:"presentFields"`
	MissingFields   []FieldReport `json:"missingFields"`
	ExtraSpecFields []FieldReport `json:"extraSpecFields"`
}

type FieldReport struct {
	FieldName string           `json:"fieldName"`
	Mandatory bool             `json:"mandatory"`
	Status    diff.FieldStatus `json:"status"`
	Reason    string           `json:"reason,omitempty"`
}

type SpecField struct {
	GoName   string
	JSONName string
	display  string
}

func (sf *SpecField) DisplayName() string {
	if sf.display != "" {
		return sf.display
	}
	if sf.JSONName != "" {
		return sf.JSONName
	}
	return sf.GoName
}

type specFieldSet struct {
	lookup  map[string]*SpecField
	display map[string]*SpecField
	list    []*SpecField
}

func newSpecFieldSet() *specFieldSet {
	return &specFieldSet{
		lookup:  make(map[string]*SpecField),
		display: make(map[string]*SpecField),
		list:    []*SpecField{},
	}
}

func (s *specFieldSet) add(goName, jsonName string) {
	if goName == "" && jsonName == "" {
		return
	}
	display := jsonName
	if display == "" {
		display = goName
	}
	spec, ok := s.display[display]
	if !ok {
		spec = &SpecField{GoName: goName, JSONName: jsonName, display: display}
		s.display[display] = spec
		s.list = append(s.list, spec)
	} else {
		if spec.GoName == "" {
			spec.GoName = goName
		}
		if spec.JSONName == "" {
			spec.JSONName = jsonName
		}
	}
	if goName != "" {
		s.lookup[normalize(goName)] = spec
	}
	if jsonName != "" {
		s.lookup[normalize(jsonName)] = spec
	}
}

func BuildReport(sdkStructs []sdk.SDKStruct, allow allowlist.Allowlist) (Report, error) {
	sdkIndex := map[string]sdk.SDKStruct{}
	for _, s := range sdkStructs {
		sdkIndex[s.QualifiedName] = s
	}

	report := Report{}
	for _, target := range Targets() {
		specFields := collectSpecFields(target.SpecType)
		for _, sdkName := range target.SDKStructs {
			sdkStruct, ok := sdkIndex[sdkName]
			if !ok {
				continue
			}
			missing := make([]FieldReport, 0)
			present := make([]FieldReport, 0)
			extra := make([]FieldReport, 0)
			matched := map[*SpecField]bool{}
			for _, field := range sdkStruct.Fields {
				if specField, ok := hasSpecField(specFields, field); ok {
					matched[specField] = true
					present = append(present, FieldReport{
						FieldName: field.Name,
						Mandatory: field.Mandatory,
						Status:    diff.FieldStatusUsed,
						Reason:    "Field is exposed in the API spec.",
					})
					continue
				}
				status, reason := classifyMissingField(allow, sdkStruct.QualifiedName, field)
				missing = append(missing, FieldReport{
					FieldName: field.Name,
					Mandatory: field.Mandatory,
					Status:    status,
					Reason:    reason,
				})
			}

			for _, specField := range specFields.list {
				if matched[specField] {
					continue
				}
				extra = append(extra, FieldReport{
					FieldName: specField.DisplayName(),
					Mandatory: false,
					Status:    diff.FieldStatusUnclassified,
					Reason:    "Field exists in the API spec but has no matching field in the tracked SDK payloads.",
				})
			}

			sort.Slice(present, func(i, j int) bool { return present[i].FieldName < present[j].FieldName })
			sort.Slice(missing, func(i, j int) bool { return missing[i].FieldName < missing[j].FieldName })
			sort.Slice(extra, func(i, j int) bool { return extra[i].FieldName < extra[j].FieldName })

			report.Structs = append(report.Structs, StructReport{
				Spec:            target.Name,
				SDKStruct:       sdkStruct.QualifiedName,
				PresentFields:   present,
				MissingFields:   missing,
				ExtraSpecFields: extra,
			})
		}
	}
	sort.Slice(report.Structs, func(i, j int) bool {
		if report.Structs[i].Spec == report.Structs[j].Spec {
			return report.Structs[i].SDKStruct < report.Structs[j].SDKStruct
		}
		return report.Structs[i].Spec < report.Structs[j].Spec
	})
	return report, nil
}

func HasActionable(report Report) bool {
	for _, s := range report.Structs {
		for _, field := range s.MissingFields {
			if field.Status == diff.FieldStatusPotentialGap || field.Status == diff.FieldStatusUnclassified {
				return true
			}
		}
		for _, field := range s.ExtraSpecFields {
			if field.Status == diff.FieldStatusPotentialGap || field.Status == diff.FieldStatusUnclassified {
				return true
			}
		}
	}
	return false
}

func collectSpecFields(t reflect.Type) *specFieldSet {
	set := newSpecFieldSet()
	visited := map[reflect.Type]bool{}
	collectFields(t, set, visited)
	return set
}

func collectFields(t reflect.Type, set *specFieldSet, visited map[reflect.Type]bool) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	if visited[t] {
		return
	}
	visited[t] = true
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		if shouldInline(field, jsonTag) {
			collectFields(field.Type, set, visited)
			continue
		}
		set.add(field.Name, jsonName(field.Tag))
	}
}

func shouldInline(field reflect.StructField, jsonTag string) bool {
	if field.Anonymous && field.Type.Kind() == reflect.Struct {
		return true
	}
	if strings.Contains(jsonTag, ",inline") && field.Type.Kind() == reflect.Struct {
		return true
	}
	return false
}

func classifyMissingField(allow allowlist.Allowlist, structName string, field sdk.SDKField) (diff.FieldStatus, string) {
	if structEntry, ok := allow.Structs[structName]; ok {
		if allowField, ok := structEntry.Fields[field.Name]; ok {
			status := mapAllowlistStatus(allowField.Status, field)
			reason := allowField.Reason
			if strings.TrimSpace(reason) == "" {
				reason = defaultReason(status, field)
			}
			return status, reason
		}
	}
	status := defaultStatus(field)
	reason := defaultReason(status, field)
	return status, reason
}

func mapAllowlistStatus(status allowlist.Status, field sdk.SDKField) diff.FieldStatus {
	switch status {
	case allowlist.StatusIntentionallyOmitted:
		return diff.FieldStatusIntentionallyOmitted
	case allowlist.StatusPotentialGap:
		return diff.FieldStatusPotentialGap
	case allowlist.StatusFutureConsideration:
		return diff.FieldStatusFutureConsideration
	case allowlist.StatusUsed:
		return diff.FieldStatusUnclassified
	default:
		return defaultStatus(field)
	}
}

func defaultStatus(field sdk.SDKField) diff.FieldStatus {
	if field.Mandatory {
		return diff.FieldStatusPotentialGap
	}
	return diff.FieldStatusFutureConsideration
}

func defaultReason(status diff.FieldStatus, field sdk.SDKField) string {
	switch status {
	case diff.FieldStatusPotentialGap:
		return "Mandatory SDK field is not exposed in the API spec."
	case diff.FieldStatusFutureConsideration:
		return "Optional SDK field is not exposed in the API spec."
	case diff.FieldStatusIntentionallyOmitted:
		return "Field intentionally omitted from the API spec."
	default:
		return "Field is not currently exposed in the API spec."
	}
}

func jsonName(tag reflect.StructTag) string {
	if tag == "" {
		return ""
	}
	jsonTag := tag.Get("json")
	if jsonTag == "" {
		return ""
	}
	name := strings.Split(jsonTag, ",")[0]
	if name == "" || name == "-" {
		return ""
	}
	return name
}
func hasSpecField(set *specFieldSet, field sdk.SDKField) (*SpecField, bool) {
	if spec, ok := set.lookup[normalize(field.Name)]; ok {
		return spec, true
	}
	if field.JSONName != "" {
		if spec, ok := set.lookup[normalize(field.JSONName)]; ok {
			return spec, true
		}
	}
	return nil, false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
