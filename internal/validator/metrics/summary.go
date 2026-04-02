package metrics

import (
	"sort"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/apispec"
)

type APISummary struct {
	Aggregate      AggregateSummary `json:"aggregate"`
	Services       []ServiceSummary `json:"services"`
	ScopeBreakdown []ScopeSummary   `json:"scopeBreakdown,omitempty"`
	TopOffenders   TopOffenders     `json:"topOffenders"`
}

type AggregateSummary struct {
	Specs                    int     `json:"specs"`
	Mappings                 int     `json:"mappings"`
	TrackedMappings          int     `json:"trackedMappings"`
	UntrackedMappings        int     `json:"untrackedMappings"`
	PresentFields            int     `json:"presentFields"`
	MissingFields            int     `json:"missingFields"`
	MandatoryPresentFields   int     `json:"mandatoryPresentFields"`
	MandatoryMissingFields   int     `json:"mandatoryMissingFields"`
	ExtraSpecFields          int     `json:"extraSpecFields"`
	OverallCoveragePercent   float64 `json:"overallCoveragePercent"`
	MandatoryCoveragePercent float64 `json:"mandatoryCoveragePercent"`
}

type ServiceSummary struct {
	Service                  string  `json:"service"`
	Specs                    int     `json:"specs"`
	Mappings                 int     `json:"mappings"`
	TrackedMappings          int     `json:"trackedMappings"`
	UntrackedMappings        int     `json:"untrackedMappings"`
	PresentFields            int     `json:"presentFields"`
	MissingFields            int     `json:"missingFields"`
	MandatoryPresentFields   int     `json:"mandatoryPresentFields"`
	MandatoryMissingFields   int     `json:"mandatoryMissingFields"`
	ExtraSpecFields          int     `json:"extraSpecFields"`
	OverallCoveragePercent   float64 `json:"overallCoveragePercent"`
	MandatoryCoveragePercent float64 `json:"mandatoryCoveragePercent"`
}

type ScopeSummary struct {
	Scope     string           `json:"scope"`
	Aggregate AggregateSummary `json:"aggregate"`
	Services  []ServiceSummary `json:"services,omitempty"`
}

type TopOffenders struct {
	MissingFields          []Offender `json:"missingFields"`
	MandatoryMissingFields []Offender `json:"mandatoryMissingFields"`
	ExtraSpecFields        []Offender `json:"extraSpecFields"`
}

type Offender struct {
	Service    string   `json:"service"`
	Spec       string   `json:"spec"`
	APISurface string   `json:"apiSurface,omitempty"`
	SDKStruct  string   `json:"sdkStruct"`
	Count      int      `json:"count"`
	FieldNames []string `json:"fieldNames,omitempty"`
}

type counter struct {
	specs                  map[string]struct{}
	mappings               int
	trackedMappings        int
	untrackedMappings      int
	presentFields          int
	missingFields          int
	mandatoryPresentFields int
	mandatoryMissingFields int
	extraSpecFields        int
}

const (
	scopeDesiredState = "desiredState"
	scopeStatusParity = "statusParity"
	scopeBroadening   = "broadening"
	scopeResponseBody = "responseBody"
)

func SummarizeAPI(report apispec.Report, topN int) APISummary {
	if topN < 0 {
		topN = 0
	}

	aggregate := newCounter()
	byService := map[string]*counter{}
	byScope := map[string]*scopeCounter{}
	missing := make([]Offender, 0, len(report.Structs))
	mandatoryMissing := make([]Offender, 0, len(report.Structs))
	extra := make([]Offender, 0, len(report.Structs))

	for _, structReport := range report.Structs {
		if structReport.APISurface == "excluded" {
			continue
		}

		aggregate.add(structReport)

		serviceCounter, ok := byService[structReport.Service]
		if !ok {
			serviceCounter = newCounter()
			byService[structReport.Service] = serviceCounter
		}
		serviceCounter.add(structReport)
		scopeName := classifyScope(structReport)
		scopeCounter := byScope[scopeName]
		if scopeCounter == nil {
			scopeCounter = newScopeCounter()
			byScope[scopeName] = scopeCounter
		}
		scopeCounter.add(structReport)

		if structReport.TrackingStatus == apispec.TrackingStatusUntracked {
			continue
		}

		if names := fieldNames(structReport.MissingFields, false); len(names) > 0 {
			missing = append(missing, newOffender(structReport, names))
		}
		if names := fieldNames(structReport.MissingFields, true); len(names) > 0 {
			mandatoryMissing = append(mandatoryMissing, newOffender(structReport, names))
		}
		if countsExtraSpecFields(structReport.APISurface) {
			if names := fieldNames(structReport.ExtraSpecFields, false); len(names) > 0 {
				extra = append(extra, newOffender(structReport, names))
			}
		}
	}

	services := make([]ServiceSummary, 0, len(byService))
	for service, counts := range byService {
		services = append(services, ServiceSummary{
			Service:                  service,
			Specs:                    len(counts.specs),
			Mappings:                 counts.mappings,
			TrackedMappings:          counts.trackedMappings,
			UntrackedMappings:        counts.untrackedMappings,
			PresentFields:            counts.presentFields,
			MissingFields:            counts.missingFields,
			MandatoryPresentFields:   counts.mandatoryPresentFields,
			MandatoryMissingFields:   counts.mandatoryMissingFields,
			ExtraSpecFields:          counts.extraSpecFields,
			OverallCoveragePercent:   coveragePercent(counts.presentFields, counts.presentFields+counts.missingFields),
			MandatoryCoveragePercent: coveragePercent(counts.mandatoryPresentFields, counts.mandatoryPresentFields+counts.mandatoryMissingFields),
		})
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Service < services[j].Service })

	return APISummary{
		Aggregate: AggregateSummary{
			Specs:                    len(aggregate.specs),
			Mappings:                 aggregate.mappings,
			TrackedMappings:          aggregate.trackedMappings,
			UntrackedMappings:        aggregate.untrackedMappings,
			PresentFields:            aggregate.presentFields,
			MissingFields:            aggregate.missingFields,
			MandatoryPresentFields:   aggregate.mandatoryPresentFields,
			MandatoryMissingFields:   aggregate.mandatoryMissingFields,
			ExtraSpecFields:          aggregate.extraSpecFields,
			OverallCoveragePercent:   coveragePercent(aggregate.presentFields, aggregate.presentFields+aggregate.missingFields),
			MandatoryCoveragePercent: coveragePercent(aggregate.mandatoryPresentFields, aggregate.mandatoryPresentFields+aggregate.mandatoryMissingFields),
		},
		Services:       services,
		ScopeBreakdown: summarizeScopes(byScope),
		TopOffenders: TopOffenders{
			MissingFields:          limitOffenders(sortOffenders(missing), topN),
			MandatoryMissingFields: limitOffenders(sortOffenders(mandatoryMissing), topN),
			ExtraSpecFields:        limitOffenders(sortOffenders(extra), topN),
		},
	}
}

func newCounter() *counter {
	return &counter{specs: make(map[string]struct{})}
}

type scopeCounter struct {
	aggregate *counter
	byService map[string]*counter
}

func newScopeCounter() *scopeCounter {
	return &scopeCounter{
		aggregate: newCounter(),
		byService: make(map[string]*counter),
	}
}

func (s *scopeCounter) add(structReport apispec.StructReport) {
	s.aggregate.add(structReport)
	serviceCounter, ok := s.byService[structReport.Service]
	if !ok {
		serviceCounter = newCounter()
		s.byService[structReport.Service] = serviceCounter
	}
	serviceCounter.add(structReport)
}

func (c *counter) add(structReport apispec.StructReport) {
	if structReport.APISurface == "excluded" {
		return
	}

	c.specs[structReport.Service+"."+structReport.Spec] = struct{}{}
	c.mappings++
	if structReport.TrackingStatus == apispec.TrackingStatusUntracked {
		c.untrackedMappings++
		return
	}

	c.trackedMappings++
	c.presentFields += len(structReport.PresentFields)
	c.missingFields += len(structReport.MissingFields)
	if countsExtraSpecFields(structReport.APISurface) {
		c.extraSpecFields += len(structReport.ExtraSpecFields)
	}
	c.mandatoryPresentFields += countMandatory(structReport.PresentFields)
	c.mandatoryMissingFields += countMandatory(structReport.MissingFields)
}

func summarizeScopes(byScope map[string]*scopeCounter) []ScopeSummary {
	scopeNames := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopeNames = append(scopeNames, scope)
	}
	sort.Slice(scopeNames, func(i, j int) bool {
		return scopeSortKey(scopeNames[i]) < scopeSortKey(scopeNames[j])
	})

	summaries := make([]ScopeSummary, 0, len(scopeNames))
	for _, scope := range scopeNames {
		counts := byScope[scope]
		summaries = append(summaries, ScopeSummary{
			Scope:     scope,
			Aggregate: newAggregateSummary(counts.aggregate),
			Services:  summarizeServiceCounts(counts.byService),
		})
	}
	return summaries
}

func summarizeServiceCounts(byService map[string]*counter) []ServiceSummary {
	services := make([]ServiceSummary, 0, len(byService))
	for service, counts := range byService {
		services = append(services, newServiceSummary(service, counts))
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Service < services[j].Service })
	return services
}

func newAggregateSummary(counts *counter) AggregateSummary {
	return AggregateSummary{
		Specs:                    len(counts.specs),
		Mappings:                 counts.mappings,
		TrackedMappings:          counts.trackedMappings,
		UntrackedMappings:        counts.untrackedMappings,
		PresentFields:            counts.presentFields,
		MissingFields:            counts.missingFields,
		MandatoryPresentFields:   counts.mandatoryPresentFields,
		MandatoryMissingFields:   counts.mandatoryMissingFields,
		ExtraSpecFields:          counts.extraSpecFields,
		OverallCoveragePercent:   coveragePercent(counts.presentFields, counts.presentFields+counts.missingFields),
		MandatoryCoveragePercent: coveragePercent(counts.mandatoryPresentFields, counts.mandatoryPresentFields+counts.mandatoryMissingFields),
	}
}

func newServiceSummary(service string, counts *counter) ServiceSummary {
	return ServiceSummary{
		Service:                  service,
		Specs:                    len(counts.specs),
		Mappings:                 counts.mappings,
		TrackedMappings:          counts.trackedMappings,
		UntrackedMappings:        counts.untrackedMappings,
		PresentFields:            counts.presentFields,
		MissingFields:            counts.missingFields,
		MandatoryPresentFields:   counts.mandatoryPresentFields,
		MandatoryMissingFields:   counts.mandatoryMissingFields,
		ExtraSpecFields:          counts.extraSpecFields,
		OverallCoveragePercent:   coveragePercent(counts.presentFields, counts.presentFields+counts.missingFields),
		MandatoryCoveragePercent: coveragePercent(counts.mandatoryPresentFields, counts.mandatoryPresentFields+counts.mandatoryMissingFields),
	}
}

func countsExtraSpecFields(apiSurface string) bool {
	return apiSurface != "status"
}

func classifyScope(structReport apispec.StructReport) string {
	switch strings.ToLower(strings.TrimSpace(structReport.APISurface)) {
	case "status":
		return scopeStatusParity
	case "responsebody":
		return scopeResponseBody
	default:
		if isDesiredStateMapping(structReport.SDKStruct) {
			return scopeDesiredState
		}
		return scopeBroadening
	}
}

func isDesiredStateMapping(sdkStruct string) bool {
	qualifiedName := strings.TrimSpace(sdkStruct)
	if qualifiedName == "" {
		return true
	}
	if dot := strings.LastIndex(qualifiedName, "."); dot >= 0 {
		qualifiedName = qualifiedName[dot+1:]
	}
	return strings.HasSuffix(qualifiedName, "Details")
}

func scopeSortKey(scope string) string {
	switch scope {
	case scopeDesiredState:
		return "0"
	case scopeStatusParity:
		return "1"
	case scopeBroadening:
		return "2"
	case scopeResponseBody:
		return "3"
	default:
		return "9:" + scope
	}
}

func countMandatory(fields []apispec.FieldReport) int {
	total := 0
	for _, field := range fields {
		if field.Mandatory {
			total++
		}
	}
	return total
}

func fieldNames(fields []apispec.FieldReport, mandatoryOnly bool) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		if mandatoryOnly && !field.Mandatory {
			continue
		}
		names = append(names, field.FieldName)
	}
	return names
}

func newOffender(structReport apispec.StructReport, fieldNames []string) Offender {
	return Offender{
		Service:    structReport.Service,
		Spec:       structReport.Spec,
		APISurface: structReport.APISurface,
		SDKStruct:  structReport.SDKStruct,
		Count:      len(fieldNames),
		FieldNames: fieldNames,
	}
}

func sortOffenders(offenders []Offender) []Offender {
	sort.Slice(offenders, func(i, j int) bool {
		if offenders[i].Count != offenders[j].Count {
			return offenders[i].Count > offenders[j].Count
		}
		if offenders[i].Service != offenders[j].Service {
			return offenders[i].Service < offenders[j].Service
		}
		if offenders[i].Spec != offenders[j].Spec {
			return offenders[i].Spec < offenders[j].Spec
		}
		if offenders[i].APISurface != offenders[j].APISurface {
			return offenders[i].APISurface < offenders[j].APISurface
		}
		return offenders[i].SDKStruct < offenders[j].SDKStruct
	})
	return offenders
}

func limitOffenders(offenders []Offender, topN int) []Offender {
	if topN == 0 || len(offenders) <= topN {
		return offenders
	}
	return offenders[:topN]
}

func coveragePercent(present int, eligible int) float64 {
	if eligible == 0 {
		return 100
	}
	return (float64(present) / float64(eligible)) * 100
}
