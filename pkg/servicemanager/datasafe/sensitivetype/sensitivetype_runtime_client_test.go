/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetype

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
)

func TestBuildSensitiveTypePolymorphicBodies(t *testing.T) {
	resource := &datasafev1beta1.SensitiveType{Spec: datasafev1beta1.SensitiveTypeSpec{
		CompartmentId: "ocid1.compartment.oc1..test",
		EntityType:    "SENSITIVE_TYPE",
		DisplayName:   "example",
		NamePattern:   "example.*",
		SearchType:    "OR",
	}}

	createBody, err := buildSensitiveTypeCreateBody(context.Background(), resource, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := createBody.(datasafesdk.CreateSensitiveTypePatternDetails); !ok {
		t.Fatalf("create body type = %T, want CreateSensitiveTypePatternDetails", createBody)
	}

	current := datasafesdk.SensitiveTypePattern{
		CompartmentId:  common.String(resource.Spec.CompartmentId),
		DisplayName:    common.String(resource.Spec.DisplayName),
		NamePattern:    common.String(resource.Spec.NamePattern),
		SearchType:     datasafesdk.SensitiveTypePatternSearchTypeOr,
		LifecycleState: datasafesdk.DiscoveryLifecycleStateActive,
	}
	updateBody, updateNeeded, err := buildSensitiveTypeUpdateBody(
		context.Background(), resource, "", datasafesdk.GetSensitiveTypeResponse{SensitiveType: current},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := updateBody.(datasafesdk.UpdateSensitiveTypePatternDetails); !ok {
		t.Fatalf("update body type = %T, want UpdateSensitiveTypePatternDetails", updateBody)
	}
	if updateNeeded {
		t.Fatal("updateNeeded = true for matching current resource")
	}

	resource.Spec.Description = "changed"
	_, updateNeeded, err = buildSensitiveTypeUpdateBody(
		context.Background(), resource, "", datasafesdk.GetSensitiveTypeResponse{SensitiveType: current},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !updateNeeded {
		t.Fatal("updateNeeded = false after mutable description change")
	}
}
