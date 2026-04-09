package metrics

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestBaselineRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	summary := APISummary{
		Aggregate: AggregateSummary{
			Specs:                    2,
			Mappings:                 3,
			TrackedMappings:          2,
			UntrackedMappings:        1,
			PresentFields:            10,
			MissingFields:            5,
			MandatoryPresentFields:   4,
			MandatoryMissingFields:   2,
			ExtraSpecFields:          3,
			OverallCoveragePercent:   66.6,
			MandatoryCoveragePercent: 66.6,
		},
		Services: []ServiceSummary{
			{
				Service:                  "functions",
				Specs:                    1,
				Mappings:                 2,
				TrackedMappings:          2,
				PresentFields:            10,
				MissingFields:            5,
				MandatoryPresentFields:   4,
				MandatoryMissingFields:   2,
				ExtraSpecFields:          3,
				OverallCoveragePercent:   66.6,
				MandatoryCoveragePercent: 66.6,
			},
		},
	}

	if err := WriteBaseline(path, summary); err != nil {
		t.Fatalf("WriteBaseline() error = %v", err)
	}

	got, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("LoadBaseline() error = %v", err)
	}

	want := BaselineFromSummary(summary)
	if got.Aggregate != want.Aggregate {
		t.Fatalf("aggregate mismatch: got %+v want %+v", got.Aggregate, want.Aggregate)
	}
	if !slices.Equal(got.Services, want.Services) {
		t.Fatalf("services mismatch: got %+v want %+v", got.Services, want.Services)
	}
}

func TestCompareSummary(t *testing.T) {
	t.Parallel()

	baseline := Baseline{
		Aggregate: AggregateSummary{
			Specs:                    10,
			Mappings:                 20,
			TrackedMappings:          18,
			UntrackedMappings:        2,
			PresentFields:            100,
			MissingFields:            50,
			MandatoryPresentFields:   40,
			MandatoryMissingFields:   10,
			ExtraSpecFields:          12,
			OverallCoveragePercent:   66.6667,
			MandatoryCoveragePercent: 80,
		},
		Services: []ServiceSummary{
			{
				Service:                  "core",
				Specs:                    5,
				Mappings:                 8,
				TrackedMappings:          8,
				UntrackedMappings:        0,
				PresentFields:            60,
				MissingFields:            20,
				MandatoryPresentFields:   20,
				MandatoryMissingFields:   4,
				ExtraSpecFields:          5,
				OverallCoveragePercent:   75,
				MandatoryCoveragePercent: 83.3333,
			},
			{
				Service:                  "functions",
				Specs:                    5,
				Mappings:                 12,
				TrackedMappings:          10,
				UntrackedMappings:        2,
				PresentFields:            40,
				MissingFields:            30,
				MandatoryPresentFields:   20,
				MandatoryMissingFields:   6,
				ExtraSpecFields:          7,
				OverallCoveragePercent:   57.1429,
				MandatoryCoveragePercent: 76.9231,
			},
		},
	}

	current := APISummary{
		Aggregate: AggregateSummary{
			Specs:                    11,
			Mappings:                 20,
			TrackedMappings:          17,
			UntrackedMappings:        3,
			PresentFields:            99,
			MissingFields:            55,
			MandatoryPresentFields:   39,
			MandatoryMissingFields:   11,
			ExtraSpecFields:          13,
			OverallCoveragePercent:   64.2857,
			MandatoryCoveragePercent: 78,
		},
		Services: []ServiceSummary{
			{
				Service:                  "core",
				Specs:                    6,
				Mappings:                 8,
				TrackedMappings:          7,
				UntrackedMappings:        1,
				PresentFields:            59,
				MissingFields:            22,
				MandatoryPresentFields:   19,
				MandatoryMissingFields:   5,
				ExtraSpecFields:          8,
				OverallCoveragePercent:   72.8395,
				MandatoryCoveragePercent: 79.1667,
			},
			{
				Service:                  "database",
				Specs:                    5,
				Mappings:                 12,
				TrackedMappings:          10,
				UntrackedMappings:        2,
				PresentFields:            40,
				MissingFields:            30,
				MandatoryPresentFields:   20,
				MandatoryMissingFields:   6,
				ExtraSpecFields:          7,
				OverallCoveragePercent:   57.1429,
				MandatoryCoveragePercent: 76.9231,
			},
		},
	}

	comparison := CompareSummary(current, baseline)
	if !comparison.HasFailures() {
		t.Fatalf("CompareSummary() expected failures, got none")
	}

	wantScope := []string{
		"aggregate specs changed from 10 to 11",
		"service \"core\" specs changed from 5 to 6",
		"service \"database\" is new in the current report",
		"service \"functions\" is missing from the current report",
	}
	for _, want := range wantScope {
		if !slices.Contains(comparison.ScopeChanges, want) {
			t.Fatalf("expected scope change %q, got %v", want, comparison.ScopeChanges)
		}
	}

	wantRegressions := []string{
		"aggregate extraSpecFields increased from 12 to 13",
		"aggregate mandatoryCoveragePercent decreased from 80.0000 to 78.0000",
		"aggregate mandatoryMissingFields increased from 10 to 11",
		"aggregate mandatoryPresentFields decreased from 40 to 39",
		"aggregate missingFields increased from 50 to 55",
		"aggregate overallCoveragePercent decreased from 66.6667 to 64.2857",
		"aggregate presentFields decreased from 100 to 99",
		"aggregate trackedMappings decreased from 18 to 17",
		"aggregate untrackedMappings increased from 2 to 3",
		"service \"core\" extraSpecFields increased from 5 to 8",
		"service \"core\" mandatoryCoveragePercent decreased from 83.3333 to 79.1667",
		"service \"core\" mandatoryMissingFields increased from 4 to 5",
		"service \"core\" mandatoryPresentFields decreased from 20 to 19",
		"service \"core\" missingFields increased from 20 to 22",
		"service \"core\" overallCoveragePercent decreased from 75.0000 to 72.8395",
		"service \"core\" presentFields decreased from 60 to 59",
		"service \"core\" trackedMappings decreased from 8 to 7",
		"service \"core\" untrackedMappings increased from 0 to 1",
	}
	for _, want := range wantRegressions {
		if !slices.Contains(comparison.Regressions, want) {
			t.Fatalf("expected regression %q, got %v", want, comparison.Regressions)
		}
	}
}
