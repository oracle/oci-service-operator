package apispec

import (
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/allowlist"
	"github.com/oracle/oci-service-operator/internal/validator/diff"
	"github.com/oracle/oci-service-operator/internal/validator/sdk"
)

const (
	TrackingStatusTracked   = "tracked"
	TrackingStatusUntracked = "untracked"

	apiSurfaceSpec     = "spec"
	apiSurfaceStatus   = "status"
	apiSurfaceExcluded = "excluded"
)

type Report struct {
	Structs []StructReport `json:"structs"`
}

type StructReport struct {
	Service         string        `json:"service"`
	Spec            string        `json:"spec"`
	APISurface      string        `json:"apiSurface,omitempty"`
	SDKStruct       string        `json:"sdkStruct"`
	TrackingStatus  string        `json:"trackingStatus,omitempty"`
	TrackingReason  string        `json:"trackingReason,omitempty"`
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
		serviceName := targetService(target)
		specFields := collectAPIFields(target.SpecType, apiSurfaceSpec)
		statusFields := (*specFieldSet)(nil)
		if target.StatusType != nil {
			statusFields = collectAPIFields(target.StatusType, apiSurfaceStatus)
		}

		if len(target.SDKMappings) == 0 {
			if coverage, ok := responseBodyCoverageForTarget(target.Name); ok {
				report.Structs = append(report.Structs, newResponseBodyStructReport(serviceName, target.Name, coverage))
				continue
			}
			reason := "Generated API surface has no mapped SDK payloads in the validator target registry."
			if reviewed := reviewedUntrackedReason(target.Name); reviewed != "" {
				reason = reviewed
			}
			report.Structs = append(report.Structs, newUntrackedStructReport(serviceName, target.Name, defaultAPISurface(specFields, statusFields), "", reason))
			continue
		}

		for _, mapping := range target.SDKMappings {
			sdkName := mapping.SDKStruct
			if mapping.Exclude {
				report.Structs = append(report.Structs, newUntrackedStructReport(serviceName, target.Name, excludedAPISurface(mapping, statusFields), sdkName, excludedMappingReason(mapping)))
				continue
			}

			sdkStruct, ok := sdkIndex[sdkName]
			if !ok {
				report.Structs = append(report.Structs, newUntrackedStructReport(serviceName, target.Name, selectAPISurfaceForMapping(mapping, statusFields), sdkName, "API coverage target references an SDK payload that is missing from the SDK seed registry."))
				continue
			}

			apiFields, apiSurface := selectAPIFieldsForSDKStruct(mapping, sdkStruct, specFields, statusFields)
			missing := make([]FieldReport, 0)
			present := make([]FieldReport, 0)
			extra := make([]FieldReport, 0)
			matched := map[*SpecField]bool{}
			for _, field := range sdkStruct.Fields {
				if specField, ok := hasAPIField(apiFields, field); ok {
					matched[specField] = true
					present = append(present, FieldReport{
						FieldName: field.Name,
						Mandatory: field.Mandatory,
						Status:    diff.FieldStatusUsed,
						Reason:    "Field is exposed in the API " + apiSurface + ".",
					})
					continue
				}
				status, reason := classifyMissingField(allow, sdkStruct.QualifiedName, field, apiSurface)
				missing = append(missing, FieldReport{
					FieldName: field.Name,
					Mandatory: field.Mandatory,
					Status:    status,
					Reason:    reason,
				})
			}

			for _, specField := range apiFields.list {
				if matched[specField] {
					continue
				}
				extra = append(extra, FieldReport{
					FieldName: specField.DisplayName(),
					Mandatory: false,
					Status:    diff.FieldStatusUnclassified,
					Reason:    "Field exists in the API " + apiSurface + " but has no matching field in the tracked SDK payloads.",
				})
			}

			sort.Slice(present, func(i, j int) bool { return present[i].FieldName < present[j].FieldName })
			sort.Slice(missing, func(i, j int) bool { return missing[i].FieldName < missing[j].FieldName })
			sort.Slice(extra, func(i, j int) bool { return extra[i].FieldName < extra[j].FieldName })

			report.Structs = append(report.Structs, StructReport{
				Service:         serviceName,
				Spec:            target.Name,
				APISurface:      apiSurface,
				SDKStruct:       sdkStruct.QualifiedName,
				TrackingStatus:  TrackingStatusTracked,
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
		if s.TrackingStatus == TrackingStatusUntracked {
			if isIntentionalUntrackedReason(s.TrackingReason) {
				continue
			}
			return true
		}
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

func newUntrackedStructReport(service, spec, apiSurface, sdkStruct, reason string) StructReport {
	return StructReport{
		Service:        service,
		Spec:           spec,
		APISurface:     apiSurface,
		SDKStruct:      sdkStruct,
		TrackingStatus: TrackingStatusUntracked,
		TrackingReason: reason,
	}
}

func targetService(target Target) string {
	if len(target.SDKMappings) > 0 {
		parts := strings.SplitN(target.SDKMappings[0].SDKStruct, ".", 2)
		if len(parts) == 2 {
			return parts[0]
		}
	}
	return path.Base(path.Dir(target.SpecType.PkgPath()))
}

func defaultAPISurface(specFields *specFieldSet, statusFields *specFieldSet) string {
	if statusFields != nil && len(specFields.list) == 0 && len(statusFields.list) > 0 {
		return apiSurfaceStatus
	}
	return apiSurfaceSpec
}

func selectAPISurfaceForMapping(mapping SDKMapping, statusFields *specFieldSet) string {
	if explicit := normalizeAPISurface(mapping.APISurface); explicit != "" {
		return explicit
	}
	return selectAPISurfaceByName(mapping.SDKStruct, statusFields)
}

func selectAPISurfaceByName(sdkName string, statusFields *specFieldSet) string {
	if statusFields == nil {
		return apiSurfaceSpec
	}
	if prefersSpecSurface(sdkName) {
		return apiSurfaceSpec
	}
	return apiSurfaceStatus
}

func selectAPIFieldsForSDKStruct(mapping SDKMapping, sdkStruct sdk.SDKStruct, specFields *specFieldSet, statusFields *specFieldSet) (*specFieldSet, string) {
	switch normalizeAPISurface(mapping.APISurface) {
	case apiSurfaceSpec:
		return specFields, apiSurfaceSpec
	case apiSurfaceStatus:
		if statusFields == nil {
			return specFields, apiSurfaceSpec
		}
		return statusFields, apiSurfaceStatus
	}

	if statusFields == nil {
		return specFields, apiSurfaceSpec
	}

	specMatches := countMatchingFields(specFields, sdkStruct.Fields)
	statusMatches := countMatchingFields(statusFields, sdkStruct.Fields)
	switch {
	case specMatches > statusMatches:
		return specFields, apiSurfaceSpec
	case statusMatches > specMatches:
		return statusFields, apiSurfaceStatus
	case prefersSpecSurface(sdkStruct.QualifiedName):
		return specFields, apiSurfaceSpec
	default:
		return statusFields, apiSurfaceStatus
	}
}

func normalizeAPISurface(apiSurface string) string {
	switch strings.ToLower(strings.TrimSpace(apiSurface)) {
	case apiSurfaceSpec:
		return apiSurfaceSpec
	case apiSurfaceStatus:
		return apiSurfaceStatus
	default:
		return ""
	}
}

func excludedAPISurface(mapping SDKMapping, statusFields *specFieldSet) string {
	if normalizeAPISurface(mapping.APISurface) != "" {
		return apiSurfaceExcluded
	}
	if statusFields == nil {
		return apiSurfaceExcluded
	}
	return apiSurfaceExcluded
}

func countMatchingFields(set *specFieldSet, fields []sdk.SDKField) int {
	if set == nil {
		return 0
	}
	matches := 0
	for _, field := range fields {
		if _, ok := hasAPIField(set, field); ok {
			matches++
		}
	}
	return matches
}

func prefersSpecSurface(sdkName string) bool {
	typeName := sdkTypeName(sdkName)
	switch {
	case strings.HasPrefix(typeName, "Create") && strings.HasSuffix(typeName, "Details"):
		return true
	case strings.HasPrefix(typeName, "Update") && strings.HasSuffix(typeName, "Details"):
		return true
	case strings.HasPrefix(typeName, "Get") && strings.HasSuffix(typeName, "Details"):
		return true
	default:
		return false
	}
}

func sdkTypeName(sdkName string) string {
	parts := strings.SplitN(strings.TrimSpace(sdkName), ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return strings.TrimSpace(sdkName)
}

func collectAPIFields(t reflect.Type, apiSurface string) *specFieldSet {
	set := newSpecFieldSet()
	visited := map[reflect.Type]bool{}
	collectFields(t, set, visited, apiSurface)
	return set
}

func collectFields(t reflect.Type, set *specFieldSet, visited map[reflect.Type]bool, apiSurface string) {
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
		if shouldSkipCoverageField(field, apiSurface) {
			continue
		}
		if shouldInline(field, jsonTag) {
			collectFields(field.Type, set, visited, apiSurface)
			continue
		}
		set.add(field.Name, jsonName(field.Tag))
	}
}

func shouldSkipCoverageField(field reflect.StructField, apiSurface string) bool {
	if apiSurface != apiSurfaceStatus {
		return false
	}
	return field.Name == "OsokStatus" && jsonName(field.Tag) == "status"
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

func classifyMissingField(allow allowlist.Allowlist, structName string, field sdk.SDKField, apiSurface string) (diff.FieldStatus, string) {
	if structEntry, ok := allow.Structs[structName]; ok {
		if allowField, ok := structEntry.Fields[field.Name]; ok {
			status := mapAllowlistStatus(allowField.Status, field)
			reason := allowField.Reason
			if strings.TrimSpace(reason) == "" {
				reason = defaultReason(status, field, apiSurface)
			}
			return status, reason
		}
	}
	status := defaultStatus(field)
	reason := defaultReason(status, field, apiSurface)
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

func defaultReason(status diff.FieldStatus, field sdk.SDKField, apiSurface string) string {
	apiTarget := "API " + apiSurface
	switch status {
	case diff.FieldStatusPotentialGap:
		return "Mandatory SDK field is not exposed in the " + apiTarget + "."
	case diff.FieldStatusFutureConsideration:
		return "Optional SDK field is not exposed in the " + apiTarget + "."
	case diff.FieldStatusIntentionallyOmitted:
		return "Field intentionally omitted from the " + apiTarget + "."
	default:
		return "Field is not currently exposed in the " + apiTarget + "."
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
func hasAPIField(set *specFieldSet, field sdk.SDKField) (*SpecField, bool) {
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
