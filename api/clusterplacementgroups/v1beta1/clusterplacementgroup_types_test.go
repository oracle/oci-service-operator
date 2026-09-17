/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	"encoding/json"
	"testing"
)

func TestClusterPlacementGroupOmittedCapabilitiesRemainOmitted(t *testing.T) {
	t.Parallel()

	resource := ClusterPlacementGroup{
		Spec: ClusterPlacementGroupSpec{
			Name:                      "example",
			ClusterPlacementGroupType: "STANDARD",
			Description:               "example",
			AvailabilityDomain:        "example-ad",
			CompartmentId:             "ocid1.compartment.oc1..example",
		},
	}

	raw, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	assertCapabilitiesOmitted(t, decoded, "spec")
	assertCapabilitiesOmitted(t, decoded, "status")
}

func assertCapabilitiesOmitted(t *testing.T, resource map[string]any, sectionName string) {
	t.Helper()

	section, ok := resource[sectionName].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", sectionName, resource[sectionName])
	}
	if capabilities, found := section["capabilities"]; found {
		t.Fatalf("%s.capabilities = %#v, want omitted", sectionName, capabilities)
	}
}
