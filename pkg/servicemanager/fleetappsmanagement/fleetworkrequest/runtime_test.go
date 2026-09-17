/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */

package fleetworkrequest

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	fleetappssdk "github.com/oracle/oci-go-sdk/v65/fleetappsmanagement"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestRecoverResourceIDSelectsPhaseAndEntity(t *testing.T) {
	workRequest := fleetappssdk.WorkRequest{Id: common.String("work-request"), Resources: []fleetappssdk.WorkRequestResource{
		{EntityType: common.String("Fleet"), ActionType: fleetappssdk.ActionTypeUpdated, Identifier: common.String("fleet-id")},
		{EntityType: common.String("Fleet_Credential"), ActionType: fleetappssdk.ActionTypeCreated, Identifier: common.String("credential-id")},
	}}
	got, err := RecoverResourceID(workRequest, shared.OSOKAsyncPhaseCreate, "fleet credential")
	if err != nil {
		t.Fatal(err)
	}
	if got != "credential-id" {
		t.Fatalf("recovered resource ID = %q, want credential-id", got)
	}
}

func TestRecoverResourceIDRejectsAmbiguousMatches(t *testing.T) {
	workRequest := fleetappssdk.WorkRequest{Id: common.String("work-request"), Resources: []fleetappssdk.WorkRequestResource{
		{EntityType: common.String("FleetResource"), ActionType: fleetappssdk.ActionTypeUpdated, Identifier: common.String("resource-a")},
		{EntityType: common.String("fleet-resource"), ActionType: fleetappssdk.ActionTypeUpdated, Identifier: common.String("resource-b")},
	}}
	if _, err := RecoverResourceID(&workRequest, shared.OSOKAsyncPhaseUpdate, "fleet resource"); err == nil {
		t.Fatal("expected ambiguous resource identifiers to fail")
	}
}
