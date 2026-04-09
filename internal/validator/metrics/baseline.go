package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Baseline struct {
	Aggregate AggregateSummary `json:"aggregate"`
	Services  []ServiceSummary `json:"services"`
}

type Comparison struct {
	ScopeChanges []string `json:"scopeChanges,omitempty"`
	Regressions  []string `json:"regressions,omitempty"`
}

func (c Comparison) HasFailures() bool {
	return len(c.ScopeChanges) > 0 || len(c.Regressions) > 0
}

func BaselineFromSummary(summary APISummary) Baseline {
	services := make([]ServiceSummary, len(summary.Services))
	copy(services, summary.Services)
	return Baseline{
		Aggregate: summary.Aggregate,
		Services:  services,
	}
}

func LoadBaseline(path string) (Baseline, error) {
	baseline := Baseline{}
	content, err := os.ReadFile(path)
	if err != nil {
		return baseline, err
	}
	if err := json.Unmarshal(content, &baseline); err != nil {
		return Baseline{}, err
	}
	sort.Slice(baseline.Services, func(i, j int) bool { return baseline.Services[i].Service < baseline.Services[j].Service })
	return baseline, nil
}

func WriteBaseline(path string, summary APISummary) error {
	if path == "" {
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
	return encoder.Encode(BaselineFromSummary(summary))
}

func CompareSummary(current APISummary, baseline Baseline) Comparison {
	comparison := Comparison{}

	if current.Aggregate.Specs != baseline.Aggregate.Specs {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("aggregate specs changed from %d to %d", baseline.Aggregate.Specs, current.Aggregate.Specs))
	}
	if current.Aggregate.Mappings != baseline.Aggregate.Mappings {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("aggregate mappings changed from %d to %d", baseline.Aggregate.Mappings, current.Aggregate.Mappings))
	}

	appendMetricRegressions(&comparison.Regressions, "aggregate", baseline.Aggregate, current.Aggregate)

	currentServices := make(map[string]ServiceSummary, len(current.Services))
	for _, service := range current.Services {
		currentServices[service.Service] = service
	}
	baselineServices := make(map[string]ServiceSummary, len(baseline.Services))
	for _, service := range baseline.Services {
		baselineServices[service.Service] = service
	}

	currentNames := sortedServiceNames(currentServices)
	baselineNames := sortedServiceNames(baselineServices)
	for _, name := range difference(baselineNames, currentNames) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q is missing from the current report", name))
	}
	for _, name := range difference(currentNames, baselineNames) {
		comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q is new in the current report", name))
	}

	for _, name := range intersection(currentNames, baselineNames) {
		currentService := currentServices[name]
		baselineService := baselineServices[name]

		if currentService.Specs != baselineService.Specs {
			comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q specs changed from %d to %d", name, baselineService.Specs, currentService.Specs))
		}
		if currentService.Mappings != baselineService.Mappings {
			comparison.ScopeChanges = append(comparison.ScopeChanges, fmt.Sprintf("service %q mappings changed from %d to %d", name, baselineService.Mappings, currentService.Mappings))
		}

		appendMetricRegressions(&comparison.Regressions, fmt.Sprintf("service %q", name), baselineService, currentService)
	}

	sort.Strings(comparison.ScopeChanges)
	sort.Strings(comparison.Regressions)
	return comparison
}

func appendMetricRegressions(messages *[]string, scope string, baseline comparableSummary, current comparableSummary) {
	if current.GetTrackedMappings() < baseline.GetTrackedMappings() {
		*messages = append(*messages, fmt.Sprintf("%s trackedMappings decreased from %d to %d", scope, baseline.GetTrackedMappings(), current.GetTrackedMappings()))
	}
	if current.GetUntrackedMappings() > baseline.GetUntrackedMappings() {
		*messages = append(*messages, fmt.Sprintf("%s untrackedMappings increased from %d to %d", scope, baseline.GetUntrackedMappings(), current.GetUntrackedMappings()))
	}
	if current.GetPresentFields() < baseline.GetPresentFields() {
		*messages = append(*messages, fmt.Sprintf("%s presentFields decreased from %d to %d", scope, baseline.GetPresentFields(), current.GetPresentFields()))
	}
	if current.GetMissingFields() > baseline.GetMissingFields() {
		*messages = append(*messages, fmt.Sprintf("%s missingFields increased from %d to %d", scope, baseline.GetMissingFields(), current.GetMissingFields()))
	}
	if current.GetMandatoryPresentFields() < baseline.GetMandatoryPresentFields() {
		*messages = append(*messages, fmt.Sprintf("%s mandatoryPresentFields decreased from %d to %d", scope, baseline.GetMandatoryPresentFields(), current.GetMandatoryPresentFields()))
	}
	if current.GetMandatoryMissingFields() > baseline.GetMandatoryMissingFields() {
		*messages = append(*messages, fmt.Sprintf("%s mandatoryMissingFields increased from %d to %d", scope, baseline.GetMandatoryMissingFields(), current.GetMandatoryMissingFields()))
	}
	if current.GetExtraSpecFields() > baseline.GetExtraSpecFields() {
		*messages = append(*messages, fmt.Sprintf("%s extraSpecFields increased from %d to %d", scope, baseline.GetExtraSpecFields(), current.GetExtraSpecFields()))
	}
	if lessFloat(current.GetOverallCoveragePercent(), baseline.GetOverallCoveragePercent()) {
		*messages = append(*messages, fmt.Sprintf("%s overallCoveragePercent decreased from %.4f to %.4f", scope, baseline.GetOverallCoveragePercent(), current.GetOverallCoveragePercent()))
	}
	if lessFloat(current.GetMandatoryCoveragePercent(), baseline.GetMandatoryCoveragePercent()) {
		*messages = append(*messages, fmt.Sprintf("%s mandatoryCoveragePercent decreased from %.4f to %.4f", scope, baseline.GetMandatoryCoveragePercent(), current.GetMandatoryCoveragePercent()))
	}
}

type comparableSummary interface {
	GetTrackedMappings() int
	GetUntrackedMappings() int
	GetPresentFields() int
	GetMissingFields() int
	GetMandatoryPresentFields() int
	GetMandatoryMissingFields() int
	GetExtraSpecFields() int
	GetOverallCoveragePercent() float64
	GetMandatoryCoveragePercent() float64
}

func (a AggregateSummary) GetTrackedMappings() int        { return a.TrackedMappings }
func (a AggregateSummary) GetUntrackedMappings() int      { return a.UntrackedMappings }
func (a AggregateSummary) GetPresentFields() int          { return a.PresentFields }
func (a AggregateSummary) GetMissingFields() int          { return a.MissingFields }
func (a AggregateSummary) GetMandatoryPresentFields() int { return a.MandatoryPresentFields }
func (a AggregateSummary) GetMandatoryMissingFields() int { return a.MandatoryMissingFields }
func (a AggregateSummary) GetExtraSpecFields() int        { return a.ExtraSpecFields }
func (a AggregateSummary) GetOverallCoveragePercent() float64 {
	return a.OverallCoveragePercent
}
func (a AggregateSummary) GetMandatoryCoveragePercent() float64 {
	return a.MandatoryCoveragePercent
}

func (s ServiceSummary) GetTrackedMappings() int        { return s.TrackedMappings }
func (s ServiceSummary) GetUntrackedMappings() int      { return s.UntrackedMappings }
func (s ServiceSummary) GetPresentFields() int          { return s.PresentFields }
func (s ServiceSummary) GetMissingFields() int          { return s.MissingFields }
func (s ServiceSummary) GetMandatoryPresentFields() int { return s.MandatoryPresentFields }
func (s ServiceSummary) GetMandatoryMissingFields() int { return s.MandatoryMissingFields }
func (s ServiceSummary) GetExtraSpecFields() int        { return s.ExtraSpecFields }
func (s ServiceSummary) GetOverallCoveragePercent() float64 {
	return s.OverallCoveragePercent
}
func (s ServiceSummary) GetMandatoryCoveragePercent() float64 {
	return s.MandatoryCoveragePercent
}

func sortedServiceNames(values map[string]ServiceSummary) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func difference(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; ok {
			continue
		}
		out = append(out, value)
	}
	return out
}

func intersection(left []string, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; ok {
			out = append(out, value)
		}
	}
	return out
}

func lessFloat(current, baseline float64) bool {
	const epsilon = 1e-9
	return current < baseline-epsilon
}
