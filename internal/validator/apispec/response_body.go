package apispec

import (
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator/diff"
)

const apiSurfaceResponseBody = "responseBody"

type responseBodyCoverage struct {
	SDKStruct string
	FieldName string
	Encoding  string
}

// The default active validator inventory currently has no response-body-only targets.
var responseBodyCoverageTargets = map[string]responseBodyCoverage{}

func responseBodyCoverageForTarget(targetName string) (responseBodyCoverage, bool) {
	coverage, ok := responseBodyCoverageTargets[strings.TrimSpace(targetName)]
	return coverage, ok
}

func newResponseBodyStructReport(service, spec string, coverage responseBodyCoverage) StructReport {
	return StructReport{
		Service:        service,
		Spec:           spec,
		APISurface:     apiSurfaceResponseBody,
		SDKStruct:      coverage.SDKStruct,
		TrackingStatus: TrackingStatusTracked,
		PresentFields: []FieldReport{
			{
				FieldName: coverage.FieldName,
				Mandatory: false,
				Status:    diff.FieldStatusUsed,
				Reason:    responseBodyCoverageReason(coverage),
			},
		},
		MissingFields:   []FieldReport{},
		ExtraSpecFields: []FieldReport{},
	}
}

func responseBodyCoverageReason(coverage responseBodyCoverage) string {
	description := "Field is covered through the OCI response body"
	if encoding := strings.TrimSpace(coverage.Encoding); encoding != "" {
		description += " (" + encoding + ")"
	}
	return description + ", not through a reusable status struct."
}
